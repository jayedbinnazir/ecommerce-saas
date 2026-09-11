// Package repository is the database/sql implementation of the notification repo.
package repository

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/notification-service/internal/notification/domain"
	"github.com/jayedbinnazir/notification-service/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, user_id, tenant_id, type, title, body, data, read_at, created_at`

func scan(row platform.Scanner) (*domain.Notification, error) {
	var n domain.Notification
	var data []byte
	if err := row.Scan(&n.ID, &n.UserID, &n.TenantID, &n.Type, &n.Title, &n.Body, &data, &n.ReadAt, &n.CreatedAt); err != nil {
		return nil, err
	}
	n.Data = data
	return &n, nil
}

func (r *Repository) Create(ctx context.Context, n *domain.Notification) error {
	data := n.Data
	if len(data) == 0 {
		data = []byte("{}")
	}
	const q = `
		INSERT INTO notifications (user_id, tenant_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + columns
	created, err := scan(r.db.QueryRowContext(ctx, q, n.UserID, n.TenantID, n.Type, n.Title, n.Body, []byte(data)))
	if err != nil {
		return err
	}
	*n = *created
	return nil
}

func (r *Repository) ListByUser(ctx context.Context, f domain.ListFilter) ([]domain.Notification, error) {
	where := "user_id = $1"
	args := []any{f.UserID}
	if f.UnreadOnly {
		where += " AND read_at IS NULL"
	}
	args = append(args, f.Page.Limit, f.Page.Offset)

	q := `SELECT ` + columns + ` FROM notifications WHERE ` + where +
		` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Notification, 0)
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	list := make([]string, len(ids))
	for i, id := range ids {
		list[i] = id.String()
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read_at = now()
		 WHERE user_id = $1 AND id = ANY($2) AND read_at IS NULL`, userID, pq.Array(list))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET read_at = now() WHERE user_id = $1 AND read_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
