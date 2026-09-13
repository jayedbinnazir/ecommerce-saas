package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type RefreshTokenRepository struct {
	db platform.DBTX
}

func NewRefreshTokenRepository(db platform.DBTX) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

var _ authdomain.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

const refreshColumns = `
	id, user_id, session_id, family_id, token_hash,
	expires_at, revoked_at, created_at, user_agent, ip_address`

func (r *RefreshTokenRepository) Create(ctx context.Context, t *authdomain.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, session_id, family_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + refreshColumns

	return r.db.QueryRowContext(ctx, q,
		t.UserID, t.SessionID, t.FamilyID, t.TokenHash, t.ExpiresAt, t.UserAgent, t.IPAddress,
	).Scan(
		&t.ID, &t.UserID, &t.SessionID, &t.FamilyID, &t.TokenHash,
		&t.ExpiresAt, &t.RevokedAt, &t.CreatedAt, &t.UserAgent, &t.IPAddress,
	)
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*authdomain.RefreshToken, error) {
	const q = `SELECT ` + refreshColumns + ` FROM refresh_tokens WHERE token_hash = $1`

	var t authdomain.RefreshToken
	err := r.db.QueryRowContext(ctx, q, hash).Scan(
		&t.ID, &t.UserID, &t.SessionID, &t.FamilyID, &t.TokenHash,
		&t.ExpiresAt, &t.RevokedAt, &t.CreatedAt, &t.UserAgent, &t.IPAddress,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, authdomain.ErrRefreshNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = $1 AND revoked_at IS NULL`, familyID)
	return err
}

func (r *RefreshTokenRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE session_id = $1 AND revoked_at IS NULL`, sessionID)
	return err
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL
		RETURNING session_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[uuid.UUID]struct{})
	sessionIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var sid uuid.UUID
		if err := rows.Scan(&sid); err != nil {
			return nil, err
		}
		if _, ok := seen[sid]; !ok {
			seen[sid] = struct{}{}
			sessionIDs = append(sessionIDs, sid)
		}
	}
	return sessionIDs, rows.Err()
}
