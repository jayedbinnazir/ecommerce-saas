// Package repository is the database/sql implementation of the store-config repo.
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/platform"
	"github.com/jayedbinnazir/order-service/internal/storeconfig/domain"
)

type Repository struct {
	db platform.DBTX
}

func New(db platform.DBTX) *Repository { return &Repository{db: db} }

var _ domain.Repository = (*Repository)(nil)

// ---------------------------------------------------------------------
// tax config
// ---------------------------------------------------------------------

func (r *Repository) GetTaxConfig(ctx context.Context, tenantID uuid.UUID) (*domain.TaxConfig, error) {
	const q = `SELECT tenant_id, tax_rate_bps, tax_inclusive, created_at, updated_at
	           FROM order_config WHERE tenant_id = $1`
	var c domain.TaxConfig
	err := r.db.QueryRowContext(ctx, q, tenantID).
		Scan(&c.TenantID, &c.TaxRateBps, &c.TaxInclusive, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.TaxConfig{TenantID: tenantID}, nil // no config = no tax
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UpsertTaxConfig(ctx context.Context, c *domain.TaxConfig) error {
	const q = `
		INSERT INTO order_config (tenant_id, tax_rate_bps, tax_inclusive)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id) DO UPDATE
		SET tax_rate_bps = EXCLUDED.tax_rate_bps, tax_inclusive = EXCLUDED.tax_inclusive
		RETURNING tenant_id, tax_rate_bps, tax_inclusive, created_at, updated_at`
	return r.db.QueryRowContext(ctx, q, c.TenantID, c.TaxRateBps, c.TaxInclusive).
		Scan(&c.TenantID, &c.TaxRateBps, &c.TaxInclusive, &c.CreatedAt, &c.UpdatedAt)
}

// ---------------------------------------------------------------------
// shipping rates
// ---------------------------------------------------------------------

const shipCols = `id, tenant_id, name, amount_cents, free_over_cents, active, created_at, updated_at`

func scanShip(row platform.Scanner) (*domain.ShippingRate, error) {
	var s domain.ShippingRate
	if err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.AmountCents, &s.FreeOverCents, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListShippingRates(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]domain.ShippingRate, error) {
	q := `SELECT ` + shipCols + ` FROM shipping_rates WHERE tenant_id = $1`
	if activeOnly {
		q += ` AND active`
	}
	q += ` ORDER BY amount_cents, name`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.ShippingRate, 0)
	for rows.Next() {
		s, err := scanShip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) GetShippingRate(ctx context.Context, tenantID, id uuid.UUID) (*domain.ShippingRate, error) {
	const q = `SELECT ` + shipCols + ` FROM shipping_rates WHERE tenant_id = $1 AND id = $2`
	s, err := scanShip(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrShippingRateNotFound
	}
	return s, err
}

func (r *Repository) CreateShippingRate(ctx context.Context, s *domain.ShippingRate) error {
	const q = `
		INSERT INTO shipping_rates (tenant_id, name, amount_cents, free_over_cents, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + shipCols
	created, err := scanShip(r.db.QueryRowContext(ctx, q, s.TenantID, s.Name, s.AmountCents, s.FreeOverCents, s.Active))
	if err != nil {
		return err
	}
	*s = *created
	return nil
}

func (r *Repository) UpdateShippingRate(ctx context.Context, s *domain.ShippingRate) error {
	const q = `
		UPDATE shipping_rates SET name = $3, amount_cents = $4, free_over_cents = $5, active = $6
		WHERE tenant_id = $1 AND id = $2
		RETURNING ` + shipCols
	updated, err := scanShip(r.db.QueryRowContext(ctx, q, s.TenantID, s.ID, s.Name, s.AmountCents, s.FreeOverCents, s.Active))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrShippingRateNotFound
	}
	if err != nil {
		return err
	}
	*s = *updated
	return nil
}

func (r *Repository) DeleteShippingRate(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM shipping_rates WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrShippingRateNotFound)
}

// ---------------------------------------------------------------------
// coupons
// ---------------------------------------------------------------------

const couponCols = `
	id, tenant_id, code, kind, value, min_order_cents, max_redemptions, redeemed_count,
	starts_at, ends_at, active, created_at, updated_at`

func scanCoupon(row platform.Scanner) (*domain.Coupon, error) {
	var c domain.Coupon
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Code, &c.Kind, &c.Value, &c.MinOrderCents, &c.MaxRedemptions, &c.RedeemedCount,
		&c.StartsAt, &c.EndsAt, &c.Active, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) ListCoupons(ctx context.Context, tenantID uuid.UUID) ([]domain.Coupon, error) {
	const q = `SELECT ` + couponCols + ` FROM coupons WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Coupon, 0)
	for rows.Next() {
		c, err := scanCoupon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) GetCoupon(ctx context.Context, tenantID, id uuid.UUID) (*domain.Coupon, error) {
	const q = `SELECT ` + couponCols + ` FROM coupons WHERE tenant_id = $1 AND id = $2`
	c, err := scanCoupon(r.db.QueryRowContext(ctx, q, tenantID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCouponNotFound
	}
	return c, err
}

func (r *Repository) GetCouponByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Coupon, error) {
	const q = `SELECT ` + couponCols + ` FROM coupons WHERE tenant_id = $1 AND code = $2`
	c, err := scanCoupon(r.db.QueryRowContext(ctx, q, tenantID, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCouponNotFound
	}
	return c, err
}

func (r *Repository) CreateCoupon(ctx context.Context, c *domain.Coupon) error {
	const q = `
		INSERT INTO coupons (tenant_id, code, kind, value, min_order_cents, max_redemptions, starts_at, ends_at, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + couponCols
	created, err := scanCoupon(r.db.QueryRowContext(ctx, q,
		c.TenantID, c.Code, c.Kind, c.Value, c.MinOrderCents, c.MaxRedemptions, c.StartsAt, c.EndsAt, c.Active))
	if err != nil {
		if platform.IsUniqueViolation(err, "coupons_tenant_code_key") {
			return domain.ErrCouponCodeTaken
		}
		return err
	}
	*c = *created
	return nil
}

func (r *Repository) UpdateCoupon(ctx context.Context, c *domain.Coupon) error {
	const q = `
		UPDATE coupons SET
			value = $3, min_order_cents = $4, max_redemptions = $5,
			starts_at = $6, ends_at = $7, active = $8
		WHERE tenant_id = $1 AND id = $2
		RETURNING ` + couponCols
	updated, err := scanCoupon(r.db.QueryRowContext(ctx, q,
		c.TenantID, c.ID, c.Value, c.MinOrderCents, c.MaxRedemptions, c.StartsAt, c.EndsAt, c.Active))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrCouponNotFound
	}
	if err != nil {
		return err
	}
	*c = *updated
	return nil
}

func (r *Repository) DeleteCoupon(ctx context.Context, tenantID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM coupons WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrCouponNotFound)
}

func (r *Repository) Redeem(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE coupons SET redeemed_count = redeemed_count + 1
		WHERE id = $1 AND (max_redemptions IS NULL OR redeemed_count < max_redemptions)`, id)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrCouponExhausted)
}
