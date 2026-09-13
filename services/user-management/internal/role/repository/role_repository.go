// Package repository holds the database/sql implementation of the role repository.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
)

type RoleRepository struct {
	db platform.DBTX
}

func NewRoleRepository(db platform.DBTX) *RoleRepository { return &RoleRepository{db: db} }

var _ domain.RoleRepository = (*RoleRepository)(nil)

const roleColumns = `id, name, description, created_at, updated_at`

func scanRole(row platform.Scanner) (*domain.Role, error) {
	var r domain.Role
	if err := row.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *RoleRepository) Create(ctx context.Context, role *domain.Role) error {
	const q = `INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING ` + roleColumns
	created, err := scanRole(r.db.QueryRowContext(ctx, q, role.Name, role.Description))
	if err != nil {
		if platform.IsUniqueViolation(err, "roles_name_key") {
			return domain.ErrRoleAlreadyExists
		}
		return err
	}
	*role = *created
	return nil
}

func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	role, err := scanRole(r.db.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM roles WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrRoleNotFound
	}
	return role, err
}

func (r *RoleRepository) GetByName(ctx context.Context, name domain.RoleName) (*domain.Role, error) {
	role, err := scanRole(r.db.QueryRowContext(ctx, `SELECT `+roleColumns+` FROM roles WHERE name = $1`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrRoleNotFound
	}
	return role, err
}

func (r *RoleRepository) List(ctx context.Context, page platform.Page) ([]domain.Role, error) {
	const q = `SELECT ` + roleColumns + ` FROM roles ORDER BY name LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Role, 0)
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *role)
	}
	return out, rows.Err()
}

func (r *RoleRepository) Update(ctx context.Context, role *domain.Role) error {
	const q = `UPDATE roles SET name = $2, description = $3 WHERE id = $1 RETURNING ` + roleColumns
	updated, err := scanRole(r.db.QueryRowContext(ctx, q, role.ID, role.Name, role.Description))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrRoleNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "roles_name_key") {
			return domain.ErrRoleAlreadyExists
		}
		return err
	}
	*role = *updated
	return nil
}

func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM roles WHERE id = $1`, id)
	if err != nil {
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrRoleInUse
		}
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrRoleNotFound)
}
