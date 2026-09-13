// Package repository holds the database/sql implementation of the membership repo.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/membership/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

const columns = `id, tenant_id, user_id, role_id, created_at, updated_at`

func scan(row platform.Scanner) (*domain.Membership, error) {
	var m domain.Membership
	if err := row.Scan(&m.ID, &m.TenantID, &m.UserID, &m.RoleID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func scanView(row platform.Scanner) (*domain.View, error) {
	var v domain.View
	if err := row.Scan(&v.ID, &v.TenantID, &v.UserID, &v.RoleID, &v.CreatedAt, &v.UpdatedAt, &v.RoleName); err != nil {
		return nil, err
	}
	return &v, nil
}

const viewSelect = `
	SELECT m.id, m.tenant_id, m.user_id, m.role_id, m.created_at, m.updated_at, r.name
	FROM memberships m
	JOIN roles r ON r.id = m.role_id`

func (r *Repository) Create(ctx context.Context, m *domain.Membership) error {
	const q = `
		INSERT INTO memberships (tenant_id, user_id, role_id)
		VALUES ($1, $2, $3)
		RETURNING ` + columns

	created, err := scan(r.db.QueryRowContext(ctx, q, m.TenantID, m.UserID, m.RoleID))
	if err != nil {
		if platform.IsUniqueViolation(err, "memberships_tenant_user_key") {
			return domain.ErrAlreadyMember
		}
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrRefInvalid
		}
		return err
	}
	*m = *created
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	m, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM memberships WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMembershipNotFound
	}
	return m, err
}

func (r *Repository) GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Membership, error) {
	m, err := scan(r.db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM memberships WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMembershipNotFound
	}
	return m, err
}

func (r *Repository) ResolveView(ctx context.Context, tenantID, userID uuid.UUID) (*domain.View, error) {
	v, err := scanView(r.db.QueryRowContext(ctx, viewSelect+` WHERE m.tenant_id = $1 AND m.user_id = $2`, tenantID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMembershipNotFound
	}
	return v, err
}

func (r *Repository) listViews(ctx context.Context, where string, arg uuid.UUID) ([]domain.View, error) {
	rows, err := r.db.QueryContext(ctx, viewSelect+where, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.View, 0)
	for rows.Next() {
		v, err := scanView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.View, error) {
	return r.listViews(ctx, ` WHERE m.tenant_id = $1 ORDER BY m.created_at`, tenantID)
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.View, error) {
	return r.listViews(ctx, ` WHERE m.user_id = $1 ORDER BY m.created_at`, userID)
}

func (r *Repository) UpdateRole(ctx context.Context, tenantID, id, roleID uuid.UUID) (*domain.Membership, error) {
	const q = `UPDATE memberships SET role_id = $3 WHERE id = $1 AND tenant_id = $2 RETURNING ` + columns
	m, err := scan(r.db.QueryRowContext(ctx, q, id, tenantID, roleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrMembershipNotFound
	}
	if err != nil {
		if platform.IsForeignKeyViolation(err) {
			return nil, domain.ErrRefInvalid
		}
		return nil, err
	}
	return m, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM memberships WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrMembershipNotFound)
}
