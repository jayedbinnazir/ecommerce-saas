// Package repository holds the database/sql implementation of the tenant repository.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, name, slug, status, owner_user_id, created_at, updated_at`

func scan(row platform.Scanner) (*domain.Tenant, error) {
	var t domain.Tenant
	if err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.OwnerUserID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func collect(rows *sql.Rows) ([]domain.Tenant, error) {
	defer rows.Close()
	out := make([]domain.Tenant, 0)
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, t *domain.Tenant) error {
	const q = `
		INSERT INTO tenants (name, slug, status, owner_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + columns

	created, err := scan(r.db.QueryRowContext(ctx, q, t.Name, t.Slug, t.Status, t.OwnerUserID))
	if err != nil {
		if platform.IsUniqueViolation(err, "tenants_slug_key") {
			return domain.ErrSlugTaken
		}
		return err
	}
	*t = *created
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM tenants WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}
	return t, err
}

func (r *Repository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	t, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM tenants WHERE slug = $1`, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTenantNotFound
	}
	return t, err
}

func (r *Repository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Tenant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+columns+` FROM tenants WHERE owner_user_id = $1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *Repository) List(ctx context.Context, page platform.Page) ([]domain.Tenant, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+columns+` FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *Repository) Update(ctx context.Context, t *domain.Tenant) error {
	const q = `
		UPDATE tenants SET name = $2, slug = $3, status = $4
		WHERE id = $1
		RETURNING ` + columns

	updated, err := scan(r.db.QueryRowContext(ctx, q, t.ID, t.Name, t.Slug, t.Status))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrTenantNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "tenants_slug_key") {
			return domain.ErrSlugTaken
		}
		return err
	}
	*t = *updated
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrTenantNotFound)
}
