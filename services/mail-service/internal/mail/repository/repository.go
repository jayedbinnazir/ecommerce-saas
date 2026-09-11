// Package repository is the database/sql implementation of the mail log.
package repository

import (
	"context"

	"github.com/jayedbinnazir/mail-service/internal/mail/domain"
	"github.com/jayedbinnazir/mail-service/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, to_addr, subject, template, status, error, created_at`

func scan(row platform.Scanner) (*domain.Mail, error) {
	var m domain.Mail
	if err := row.Scan(&m.ID, &m.ToAddr, &m.Subject, &m.Template, &m.Status, &m.Error, &m.CreatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) Create(ctx context.Context, m *domain.Mail) error {
	const q = `
		INSERT INTO mails (to_addr, subject, template, status, error)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + columns
	created, err := scan(r.db.QueryRowContext(ctx, q, m.ToAddr, m.Subject, m.Template, m.Status, m.Error))
	if err != nil {
		return err
	}
	*m = *created
	return nil
}

func (r *Repository) ListByRecipient(ctx context.Context, toAddr string, limit int) ([]domain.Mail, error) {
	const q = `SELECT ` + columns + ` FROM mails WHERE to_addr = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.QueryContext(ctx, q, toAddr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Mail, 0)
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}
