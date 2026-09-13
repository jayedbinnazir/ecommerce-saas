// Package repository holds the database/sql implementations of the auth repos.
package repository

import (
	"context"
	"database/sql"
	"errors"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type IdentityRepository struct {
	db platform.DBTX
}

func NewIdentityRepository(db platform.DBTX) *IdentityRepository {
	return &IdentityRepository{db: db}
}

var _ authdomain.IdentityRepository = (*IdentityRepository)(nil)

const identityColumns = `id, user_id, provider, provider_subject, provider_email, created_at, updated_at`

func (r *IdentityRepository) GetByProviderSubject(ctx context.Context, provider authdomain.Provider, subject string) (*authdomain.AuthIdentity, error) {
	const q = `SELECT ` + identityColumns + ` FROM auth_identities WHERE provider = $1 AND provider_subject = $2`

	var i authdomain.AuthIdentity
	err := r.db.QueryRowContext(ctx, q, provider, subject).Scan(
		&i.ID, &i.UserID, &i.Provider, &i.ProviderSubject, &i.ProviderEmail, &i.CreatedAt, &i.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, authdomain.ErrIdentityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *IdentityRepository) Create(ctx context.Context, i *authdomain.AuthIdentity) error {
	const q = `
		INSERT INTO auth_identities (user_id, provider, provider_subject, provider_email)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + identityColumns

	return r.db.QueryRowContext(ctx, q, i.UserID, i.Provider, i.ProviderSubject, i.ProviderEmail).Scan(
		&i.ID, &i.UserID, &i.Provider, &i.ProviderSubject, &i.ProviderEmail, &i.CreatedAt, &i.UpdatedAt,
	)
}
