// Package repository is the database/sql implementation of the review repo.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/review/domain"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, tenant_id, product_id, customer_id, rating, title, body, created_at, updated_at`

func scan(row platform.Scanner) (*domain.Review, error) {
	var r domain.Review
	if err := row.Scan(&r.ID, &r.TenantID, &r.ProductID, &r.CustomerID, &r.Rating, &r.Title, &r.Body, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Repository) Upsert(ctx context.Context, rv *domain.Review) error {
	const q = `
		INSERT INTO product_reviews (tenant_id, product_id, customer_id, rating, title, body)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (product_id, customer_id) DO UPDATE
		SET rating = EXCLUDED.rating, title = EXCLUDED.title, body = EXCLUDED.body
		RETURNING ` + columns

	created, err := scan(r.db.QueryRowContext(ctx, q, rv.TenantID, rv.ProductID, rv.CustomerID, rv.Rating, rv.Title, rv.Body))
	if err != nil {
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrReviewNotFound // bad product_id
		}
		return err
	}
	*rv = *created
	return nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Review, error) {
	const q = `SELECT ` + columns + ` FROM product_reviews WHERE tenant_id = $1 AND id = $2`
	rv, err := scan(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrReviewNotFound
	}
	return rv, err
}

func (r *Repository) ListByProduct(ctx context.Context, productID uuid.UUID, page platform.Page) ([]domain.Review, error) {
	const q = `SELECT ` + columns + ` FROM product_reviews WHERE product_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, productID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Review, 0)
	for rows.Next() {
		rv, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rv)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM product_reviews WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrReviewNotFound)
}

func (r *Repository) Summary(ctx context.Context, productID uuid.UUID) (domain.Summary, error) {
	const q = `SELECT coalesce(avg(rating), 0), count(*) FROM product_reviews WHERE product_id = $1`
	var s domain.Summary
	err := r.db.QueryRowContext(ctx, q, productID).Scan(&s.Average, &s.Count)
	return s, err
}

// SummaryByProducts aggregates in one query (no N+1 for a product list).
func (r *Repository) SummaryByProducts(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID]domain.Summary, error) {
	out := make(map[uuid.UUID]domain.Summary, len(productIDs))
	if len(productIDs) == 0 {
		return out, nil
	}

	ids := make([]string, len(productIDs))
	for i, id := range productIDs {
		ids[i] = id.String()
	}

	const q = `
		SELECT product_id, avg(rating), count(*)
		FROM product_reviews
		WHERE product_id = ANY($1)
		GROUP BY product_id`
	rows, err := r.db.QueryContext(ctx, q, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pid uuid.UUID
		var s domain.Summary
		if err := rows.Scan(&pid, &s.Average, &s.Count); err != nil {
			return nil, err
		}
		out[pid] = s
	}
	return out, rows.Err()
}
