package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RefreshToken is one issued refresh token, stored only as a SHA-256 hash.
// Tokens form a "family" (FamilyID): rotating a token creates a new row in the
// same family and revokes the old one. If a already-revoked token is presented
// again, the whole family is revoked (reuse detection).
// Table "refresh_tokens".
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"         db:"id"`
	UserID    uuid.UUID  `json:"user_id"    db:"user_id"`
	SessionID uuid.UUID  `json:"session_id" db:"session_id"`
	FamilyID  uuid.UUID  `json:"family_id"  db:"family_id"`
	TokenHash string     `json:"-"          db:"token_hash"` // UNIQUE
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UserAgent *string    `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress *string    `json:"ip_address,omitempty" db:"ip_address"`
}

func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeByID(ctx context.Context, id uuid.UUID) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	// RevokeAllForUser revokes every not-yet-revoked refresh token belonging to
	// the user (every device/session) and returns the distinct session ids
	// that were revoked, so the caller can also drop their Redis sessions.
	// Used by change-password and reset-password.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
