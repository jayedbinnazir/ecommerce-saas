// Command migration is a minimal forward/backward SQL migrator.
//
//	go run ./cmd/migration up          # apply all pending migrations
//	go run ./cmd/migration down         # roll back the most recent migration
//	go run ./cmd/migration down 3       # roll back the last 3
//	go run ./cmd/migration status       # print applied / pending state
//
// Files live in ./migrations, named <version>_<name>.up.sql / .down.sql.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/lib/pq"

	"github.com/jayedbinnazir/cart-service/internal/config"
)

const migrationsDir = "./migrations"

type migration struct {
	version int64
	name    string
	upSQL   string
	downSQL string
}

func main() {
	log.SetFlags(0)

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("usage: migration <up|down|status> [n]")
	}
	cmd := args[0]

	cfg, err := config.ConfigLoader()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := open(cfg.Database)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	if err := ensureSchemaTable(db); err != nil {
		log.Fatalf("init schema table: %v", err)
	}

	all, err := loadMigrations()
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}

	applied, err := appliedVersions(db)
	if err != nil {
		log.Fatalf("read applied migrations: %v", err)
	}

	switch cmd {
	case "up":
		runUp(db, all, applied)
	case "down":
		steps := 1
		if len(args) > 1 {
			if steps, err = strconv.Atoi(args[1]); err != nil || steps < 1 {
				log.Fatalf("invalid step count: %q", args[1])
			}
		}
		runDown(db, all, applied, steps)
	case "status":
		printStatus(all, applied)
	default:
		log.Fatalf("unknown command: %q", cmd)
	}
}

func open(c config.DatabaseConfig) (*sql.DB, error) {
	ssl := "disable"
	if c.DBSSL {
		ssl = "require"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, ssl)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

func ensureSchemaTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    BIGINT PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func loadMigrations() ([]migration, error) {
	entries, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no *.up.sql files in %s", migrationsDir)
	}

	out := make([]migration, 0, len(entries))
	for _, up := range entries {
		base := strings.TrimSuffix(filepath.Base(up), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad migration filename: %s", up)
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad version in %s: %w", up, err)
		}
		upSQL, err := os.ReadFile(up)
		if err != nil {
			return nil, err
		}
		downSQL, err := os.ReadFile(filepath.Join(migrationsDir, base+".down.sql"))
		if err != nil {
			return nil, fmt.Errorf("missing down file for %s: %w", base, err)
		}
		out = append(out, migration{version, parts[1], string(upSQL), string(downSQL)})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func appliedVersions(db *sql.DB) (map[int64]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func runUp(db *sql.DB, all []migration, applied map[int64]bool) {
	ran := 0
	for _, m := range all {
		if applied[m.version] {
			continue
		}
		log.Printf("↑ applying %d_%s", m.version, m.name)
		if err := exec(db, m.upSQL,
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.version, m.name); err != nil {
			log.Fatalf("apply %d_%s: %v", m.version, m.name, err)
		}
		ran++
	}
	if ran == 0 {
		log.Println("nothing to do — database is up to date")
		return
	}
	log.Printf("done — applied %d migration(s)", ran)
}

func runDown(db *sql.DB, all []migration, applied map[int64]bool, steps int) {
	ran := 0
	for i := len(all) - 1; i >= 0 && ran < steps; i-- {
		m := all[i]
		if !applied[m.version] {
			continue
		}
		log.Printf("↓ reverting %d_%s", m.version, m.name)
		if err := exec(db, m.downSQL,
			`DELETE FROM schema_migrations WHERE version = $1`, m.version); err != nil {
			log.Fatalf("revert %d_%s: %v", m.version, m.name, err)
		}
		ran++
	}
	if ran == 0 {
		log.Println("nothing to do — no applied migrations to revert")
		return
	}
	log.Printf("done — reverted %d migration(s)", ran)
}

func exec(db *sql.DB, body, bookkeeping string, bookArgs ...any) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(body); err != nil {
		return err
	}
	if _, err := tx.Exec(bookkeeping, bookArgs...); err != nil {
		return err
	}
	return tx.Commit()
}

func printStatus(all []migration, applied map[int64]bool) {
	log.Println("VERSION       STATUS   NAME")
	for _, m := range all {
		status := "pending"
		if applied[m.version] {
			status = "applied"
		}
		log.Printf("%-13d %-8s %s", m.version, status, m.name)
	}
}
