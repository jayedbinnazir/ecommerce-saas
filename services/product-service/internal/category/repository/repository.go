// Package repository is the database/sql implementation of the category repo.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jayedbinnazir/product-service/internal/category/domain"
	"github.com/jayedbinnazir/product-service/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, tenant_id, parent_id, name, slug, position, created_at, updated_at`

func scan(row platform.Scanner) (*domain.Category, error) {
	var c domain.Category
	if err := row.Scan(&c.ID, &c.TenantID, &c.ParentID, &c.Name, &c.Slug, &c.Position, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) Create(ctx context.Context, c *domain.Category) error {
	const q = `
		INSERT INTO categories (tenant_id, parent_id, name, slug, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + columns

	created, err := scan(r.db.QueryRowContext(ctx, q, c.TenantID, c.ParentID, c.Name, c.Slug, c.Position))
	if err != nil {
		if platform.IsUniqueViolation(err, "categories_tenant_slug_key") {
			return domain.ErrSlugTaken
		}
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrCategoryNotFound // bad parent_id
		}
		return err
	}
	*c = *created
	return nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	const q = `SELECT ` + columns + ` FROM categories WHERE tenant_id = $1 AND id = $2`
	c, err := scan(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCategoryNotFound
	}
	return c, err
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.Category, error) {
	const q = `SELECT ` + columns + ` FROM categories WHERE tenant_id = $1 ORDER BY position, name`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Category, 0)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, c *domain.Category) error {
	const q = `
		UPDATE categories SET parent_id = $3, name = $4, slug = $5, position = $6
		WHERE tenant_id = $1 AND id = $2
		RETURNING ` + columns

	updated, err := scan(r.db.QueryRowContext(ctx, q, c.TenantID, c.ID, c.ParentID, c.Name, c.Slug, c.Position))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrCategoryNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "categories_tenant_slug_key") {
			return domain.ErrSlugTaken
		}
		return err
	}
	*c = *updated
	return nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrCategoryNotFound)
}

func (r *Repository) Exists(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM categories WHERE tenant_id = $1 AND id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, tenantID, id).Scan(&exists)
	return exists, err
}

func (r *Repository) CountExisting(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	list := make([]string, len(ids))
	for i, id := range ids {
		list[i] = id.String()
	}
	const q = `SELECT count(*) FROM categories WHERE tenant_id = $1 AND id = ANY($2)`
	var n int
	err := r.db.QueryRowContext(ctx, q, tenantID, pq.Array(list)).Scan(&n)
	return n, err
}
