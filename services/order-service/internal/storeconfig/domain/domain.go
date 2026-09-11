// Package domain holds the per-tenant checkout configuration: tax rate, shipping
// rates, and discount coupons.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ---- tax ----

// TaxConfig is one row per tenant (defaults to no tax).
type TaxConfig struct {
	TenantID     uuid.UUID `json:"tenant_id"     db:"tenant_id"`
	TaxRateBps   int       `json:"tax_rate_bps"  db:"tax_rate_bps"` // basis points; 875 = 8.75%
	TaxInclusive bool      `json:"tax_inclusive" db:"tax_inclusive"`
	CreatedAt    time.Time `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}

// ---- shipping ----

// ShippingRate is a delivery option the shopper picks at checkout.
type ShippingRate struct {
	ID            uuid.UUID `json:"id"             db:"id"`
	TenantID      uuid.UUID `json:"tenant_id"      db:"tenant_id"`
	Name          string    `json:"name"           db:"name"`
	AmountCents   int64     `json:"amount_cents"   db:"amount_cents"`
	FreeOverCents *int64    `json:"free_over_cents,omitempty" db:"free_over_cents"` // free when order >= this
	Active        bool      `json:"active"          db:"active"`
	CreatedAt     time.Time `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"     db:"updated_at"`
}

// CostFor returns the shipping charge given the order value after any discount.
func (r ShippingRate) CostFor(orderCents int64) int64 {
	if r.FreeOverCents != nil && orderCents >= *r.FreeOverCents {
		return 0
	}
	return r.AmountCents
}

// ---- coupons ----

type CouponKind string

const (
	CouponPercent CouponKind = "PERCENT" // value is a whole percent (1-100)
	CouponFixed   CouponKind = "FIXED"   // value is a fixed amount in cents
)

func (k CouponKind) Valid() bool { return k == CouponPercent || k == CouponFixed }

// Coupon is a discount code.
type Coupon struct {
	ID             uuid.UUID  `json:"id"              db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"       db:"tenant_id"`
	Code           string     `json:"code"            db:"code"`
	Kind           CouponKind `json:"kind"            db:"kind"`
	Value          int64      `json:"value"           db:"value"`
	MinOrderCents  int64      `json:"min_order_cents" db:"min_order_cents"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty" db:"max_redemptions"`
	RedeemedCount  int        `json:"redeemed_count"  db:"redeemed_count"`
	StartsAt       *time.Time `json:"starts_at,omitempty" db:"starts_at"`
	EndsAt         *time.Time `json:"ends_at,omitempty"   db:"ends_at"`
	Active         bool       `json:"active"          db:"active"`
	CreatedAt      time.Time  `json:"created_at"      db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"      db:"updated_at"`
}

// Redeemable reports whether the coupon can be applied to an order worth
// subtotalCents at time now.
func (c Coupon) Redeemable(subtotalCents int64, now time.Time) error {
	if !c.Active {
		return ErrCouponInactive
	}
	if c.StartsAt != nil && now.Before(*c.StartsAt) {
		return ErrCouponNotStarted
	}
	if c.EndsAt != nil && now.After(*c.EndsAt) {
		return ErrCouponExpired
	}
	if c.MaxRedemptions != nil && c.RedeemedCount >= *c.MaxRedemptions {
		return ErrCouponExhausted
	}
	if subtotalCents < c.MinOrderCents {
		return ErrCouponMinOrder
	}
	return nil
}

// DiscountFor returns the discount in cents, capped at the subtotal.
func (c Coupon) DiscountFor(subtotalCents int64) int64 {
	var d int64
	switch c.Kind {
	case CouponPercent:
		d = subtotalCents * c.Value / 100
	default: // FIXED
		d = c.Value
	}
	if d > subtotalCents {
		d = subtotalCents
	}
	return d
}

// ---- repository ----

type Repository interface {
	// tax
	GetTaxConfig(ctx context.Context, tenantID uuid.UUID) (*TaxConfig, error)
	UpsertTaxConfig(ctx context.Context, c *TaxConfig) error

	// shipping
	ListShippingRates(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]ShippingRate, error)
	GetShippingRate(ctx context.Context, tenantID, id uuid.UUID) (*ShippingRate, error)
	CreateShippingRate(ctx context.Context, r *ShippingRate) error
	UpdateShippingRate(ctx context.Context, r *ShippingRate) error
	DeleteShippingRate(ctx context.Context, tenantID, id uuid.UUID) error

	// coupons
	ListCoupons(ctx context.Context, tenantID uuid.UUID) ([]Coupon, error)
	GetCoupon(ctx context.Context, tenantID, id uuid.UUID) (*Coupon, error)
	GetCouponByCode(ctx context.Context, tenantID uuid.UUID, code string) (*Coupon, error)
	CreateCoupon(ctx context.Context, c *Coupon) error
	UpdateCoupon(ctx context.Context, c *Coupon) error
	DeleteCoupon(ctx context.Context, tenantID, id uuid.UUID) error
	// Redeem bumps redeemed_count, failing (0 rows) if the cap is now hit.
	Redeem(ctx context.Context, id uuid.UUID) error
}
