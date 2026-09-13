package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken is one issued password-reset request, stored only as a
// SHA-256 hash (same pattern as RefreshToken.TokenHash). Single-use: UsedAt is
// set the moment it is consumed, and a used or expired token is rejected.
// Table "password_reset_tokens".
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id"         db:"id"`
	UserID    uuid.UUID  `json:"user_id"    db:"user_id"`
	TokenHash string     `json:"-"          db:"token_hash"` // UNIQUE
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

func (t *PasswordResetToken) IsUsed() bool { return t.UsedAt != nil }
func (t *PasswordResetToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

type PasswordResetRepository interface {
	Create(ctx context.Context, token *PasswordResetToken) error
	// GetByHashForUpdate locks the row (SELECT ... FOR UPDATE) so a concurrent
	// double-submit of the same token can't both pass the not-used-yet check.
	GetByHashForUpdate(ctx context.Context, hash string) (*PasswordResetToken, error)
	// MarkUsed is a WHERE-guarded single-use UPDATE — defense in depth on top
	// of the row lock above.
	MarkUsed(ctx context.Context, id uuid.UUID) error
}
