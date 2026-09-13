package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
)

type AddressRepository struct {
	db platform.DBTX
}

func NewAddressRepository(db platform.DBTX) *AddressRepository {
	return &AddressRepository{db: db}
}

var _ domain.AddressRepository = (*AddressRepository)(nil)

const addressColumns = `
	id, user_id, label, recipient_name, phone,
	address_line_1, address_line_2, city, state, postal_code, country,
	is_default_shipping, is_default_billing, created_at, updated_at`

func scanAddress(row platform.Scanner) (*domain.Address, error) {
	var a domain.Address
	err := row.Scan(
		&a.ID, &a.UserID, &a.Label, &a.RecipientName, &a.Phone,
		&a.AddressLine1, &a.AddressLine2, &a.City, &a.State, &a.PostalCode, &a.Country,
		&a.IsDefaultShipping, &a.IsDefaultBilling, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) Create(ctx context.Context, a *domain.Address) error {
	const q = `
		INSERT INTO addresses (
			user_id, label, recipient_name, phone,
			address_line_1, address_line_2, city, state, postal_code, country,
			is_default_shipping, is_default_billing
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING ` + addressColumns

	created, err := scanAddress(r.db.QueryRowContext(ctx, q,
		a.UserID, a.Label, a.RecipientName, a.Phone,
		a.AddressLine1, a.AddressLine2, a.City, a.State, a.PostalCode, a.Country,
		a.IsDefaultShipping, a.IsDefaultBilling,
	))
	if err != nil {
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrUserNotFound
		}
		return err
	}
	*a = *created
	return nil
}

func (r *AddressRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Address, error) {
	const q = `SELECT ` + addressColumns + ` FROM addresses WHERE id = $1`
	a, err := scanAddress(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrAddressNotFound
	}
	return a, err
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	const q = `SELECT ` + addressColumns + ` FROM addresses WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Address, 0)
	for rows.Next() {
		a, err := scanAddress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AddressRepository) Update(ctx context.Context, a *domain.Address) error {
	const q = `
		UPDATE addresses SET
			label = $2, recipient_name = $3, phone = $4,
			address_line_1 = $5, address_line_2 = $6, city = $7,
			state = $8, postal_code = $9, country = $10,
			is_default_shipping = $11, is_default_billing = $12
		WHERE id = $1
		RETURNING ` + addressColumns

	updated, err := scanAddress(r.db.QueryRowContext(ctx, q,
		a.ID, a.Label, a.RecipientName, a.Phone,
		a.AddressLine1, a.AddressLine2, a.City, a.State, a.PostalCode, a.Country,
		a.IsDefaultShipping, a.IsDefaultBilling,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrAddressNotFound
	}
	if err != nil {
		return err
	}
	*a = *updated
	return nil
}

func (r *AddressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM addresses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrAddressNotFound)
}

func (r *AddressRepository) ClearDefaultShipping(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE addresses SET is_default_shipping = false WHERE user_id = $1 AND is_default_shipping`, userID)
	return err
}

func (r *AddressRepository) ClearDefaultBilling(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE addresses SET is_default_billing = false WHERE user_id = $1 AND is_default_billing`, userID)
	return err
}
