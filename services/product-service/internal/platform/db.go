package platform

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
)

// DBTX is satisfied by both *sql.DB and *sql.Tx so repositories can run inside a
// caller-supplied transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Scanner is the row-scan surface common to *sql.Row and *sql.Rows.
type Scanner interface{ Scan(dest ...any) error }

// IsUniqueViolation reports whether err is a Postgres unique-constraint failure,
// optionally scoped to a specific constraint name.
func IsUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code.Name() != "unique_violation" {
		return false
	}
	return constraint == "" || pqErr.Constraint == constraint
}

// IsForeignKeyViolation reports whether err is a Postgres FK-constraint failure.
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation"
}

// AffectedOrNotFound returns notFound when the statement touched no rows.
func AffectedOrNotFound(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

// RunInTx executes fn inside a single transaction, committing on success and
// rolling back on error or panic.
func RunInTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	return fn(tx)
}
