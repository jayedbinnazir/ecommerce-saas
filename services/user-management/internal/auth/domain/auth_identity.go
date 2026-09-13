package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Provider is an external identity provider we accept logins from.
type Provider string

const (
	ProviderGoogle   Provider = "google"
	ProviderFacebook Provider = "facebook"
)

func ParseProvider(s string) (Provider, bool) {
	switch Provider(s) {
	case ProviderGoogle:
		return ProviderGoogle, true
	case ProviderFacebook:
		return ProviderFacebook, true
	default:
		return "", false
	}
}

// AuthIdentity links a platform user to their account at an external provider.
// Table "auth_identities", UNIQUE (provider, provider_subject).
type AuthIdentity struct {
	ID              uuid.UUID `json:"id"               db:"id"`
	UserID          uuid.UUID `json:"user_id"          db:"user_id"` // REFERENCES users(id) ON DELETE CASCADE
	Provider        Provider  `json:"provider"         db:"provider"`
	ProviderSubject string    `json:"provider_subject" db:"provider_subject"` // the provider's stable user id ("sub")
	ProviderEmail   *string   `json:"provider_email,omitempty" db:"provider_email"`
	CreatedAt       time.Time `json:"created_at"       db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"       db:"updated_at"`
}

type IdentityRepository interface {
	// GetByProviderSubject returns the identity for (provider, subject) or
	// ErrIdentityNotFound.
	GetByProviderSubject(ctx context.Context, provider Provider, subject string) (*AuthIdentity, error)
	Create(ctx context.Context, identity *AuthIdentity) error
}
