// Package repository holds the database/sql implementations of the permission
// module repositories.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/permission/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type PermissionRepository struct {
	db platform.DBTX
}

func NewPermissionRepository(db platform.DBTX) *PermissionRepository {
	return &PermissionRepository{db: db}
}

var _ domain.PermissionRepository = (*PermissionRepository)(nil)

const permissionColumns = `id, name, description, created_at, updated_at`

func scanPermission(row platform.Scanner) (*domain.Permission, error) {
	var p domain.Permission
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func collectPermissions(rows *sql.Rows) ([]domain.Permission, error) {
	defer rows.Close()
	out := make([]domain.Permission, 0)
	for rows.Next() {
		p, err := scanPermission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PermissionRepository) Create(ctx context.Context, p *domain.Permission) error {
	const q = `INSERT INTO permissions (name, description) VALUES ($1, $2) RETURNING ` + permissionColumns
	created, err := scanPermission(r.db.QueryRowContext(ctx, q, p.Name, p.Description))
	if err != nil {
		if platform.IsUniqueViolation(err, "permissions_name_key") {
			return domain.ErrPermissionAlreadyExists
		}
		return err
	}
	*p = *created
	return nil
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	p, err := scanPermission(r.db.QueryRowContext(ctx, `SELECT `+permissionColumns+` FROM permissions WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPermissionNotFound
	}
	return p, err
}

func (r *PermissionRepository) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	p, err := scanPermission(r.db.QueryRowContext(ctx, `SELECT `+permissionColumns+` FROM permissions WHERE name = $1`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPermissionNotFound
	}
	return p, err
}

func (r *PermissionRepository) List(ctx context.Context, page platform.Page) ([]domain.Permission, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+permissionColumns+` FROM permissions ORDER BY name LIMIT $1 OFFSET $2`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	return collectPermissions(rows)
}

func (r *PermissionRepository) Update(ctx context.Context, p *domain.Permission) error {
	const q = `UPDATE permissions SET name = $2, description = $3 WHERE id = $1 RETURNING ` + permissionColumns
	updated, err := scanPermission(r.db.QueryRowContext(ctx, q, p.ID, p.Name, p.Description))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrPermissionNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "permissions_name_key") {
			return domain.ErrPermissionAlreadyExists
		}
		return err
	}
	*p = *updated
	return nil
}

func (r *PermissionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM permissions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrPermissionNotFound)
}
