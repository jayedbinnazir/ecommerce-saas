// Package repository holds the database/sql (lib/pq) implementations of the
// user-module domain repositories.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
)

type UserRepository struct {
	db platform.DBTX
}

func NewUserRepository(db platform.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

var _ domain.UserRepository = (*UserRepository)(nil)

const userColumns = `id, name, email, phone, password_hash, is_super_admin, created_at, updated_at`

func scanUser(row platform.Scanner) (*domain.User, error) {
	var u domain.User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.IsSuperAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (name, email, phone, password_hash, is_super_admin)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + userColumns

	created, err := scanUser(r.db.QueryRowContext(ctx, q, u.Name, u.Email, u.Phone, u.PasswordHash, u.IsSuperAdmin))
	if err != nil {
		if platform.IsUniqueViolation(err, "users_email_key") {
			return domain.ErrUserAlreadyExists
		}
		return err
	}
	*u = *created
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE email = $1`
	u, err := scanUser(r.db.QueryRowContext(ctx, q, email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (r *UserRepository) List(ctx context.Context, page platform.Page) ([]domain.User, error) {
	const q = `SELECT ` + userColumns + ` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, q, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	const q = `
		UPDATE users
		SET name = $2, email = $3, phone = $4, password_hash = $5, is_super_admin = $6
		WHERE id = $1
		RETURNING ` + userColumns

	updated, err := scanUser(r.db.QueryRowContext(ctx, q, u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.IsSuperAdmin))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	if err != nil {
		if platform.IsUniqueViolation(err, "users_email_key") {
			return domain.ErrUserAlreadyExists
		}
		return err
	}
	*u = *updated
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrUserNotFound)
}
