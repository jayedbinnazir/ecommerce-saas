package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type PasswordResetRepository struct {
	db platform.DBTX
}

func NewPasswordResetRepository(db platform.DBTX) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

var _ authdomain.PasswordResetRepository = (*PasswordResetRepository)(nil)

const resetColumns = `id, user_id, token_hash, expires_at, used_at, created_at`

func (r *PasswordResetRepository) Create(ctx context.Context, t *authdomain.PasswordResetToken) error {
	const q = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING ` + resetColumns

	return r.db.QueryRowContext(ctx, q, t.UserID, t.TokenHash, t.ExpiresAt).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
}

func (r *PasswordResetRepository) GetByHashForUpdate(ctx context.Context, hash string) (*authdomain.PasswordResetToken, error) {
	const q = `SELECT ` + resetColumns + ` FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`

	var t authdomain.PasswordResetToken
	err := r.db.QueryRowContext(ctx, q, hash).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, authdomain.ErrResetTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, authdomain.ErrResetTokenInvalid)
}
