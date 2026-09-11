// Package dto holds request/response payloads for the store-config module.
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/storeconfig/domain"
)

// ---- tax ----

type TaxConfigRequest struct {
	TaxRateBps   int  `json:"tax_rate_bps" binding:"gte=0,lte=10000"`
	TaxInclusive bool `json:"tax_inclusive"`
}

type TaxConfigResponse struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	TaxRateBps   int       `json:"tax_rate_bps"`
	TaxInclusive bool      `json:"tax_inclusive"`
}

func FromTaxConfig(c *domain.TaxConfig) TaxConfigResponse {
	return TaxConfigResponse{TenantID: c.TenantID, TaxRateBps: c.TaxRateBps, TaxInclusive: c.TaxInclusive}
}

// ---- shipping ----

type CreateShippingRateRequest struct {
	Name          string `json:"name" binding:"required,max=120"`
	AmountCents   int64  `json:"amount_cents" binding:"gte=0"`
	FreeOverCents *int64 `json:"free_over_cents" binding:"omitempty,gte=0"`
	Active        *bool  `json:"active"`
}

type UpdateShippingRateRequest struct {
	Name          *string `json:"name" binding:"omitempty,max=120"`
	AmountCents   *int64  `json:"amount_cents" binding:"omitempty,gte=0"`
	FreeOverCents *int64  `json:"free_over_cents" binding:"omitempty,gte=0"`
	Active        *bool   `json:"active"`
}

func (r UpdateShippingRateRequest) IsEmpty() bool {
	return r.Name == nil && r.AmountCents == nil && r.FreeOverCents == nil && r.Active == nil
}

type ShippingRateResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	AmountCents   int64     `json:"amount_cents"`
	FreeOverCents *int64    `json:"free_over_cents,omitempty"`
	Active        bool      `json:"active"`
}

func FromShippingRate(r *domain.ShippingRate) ShippingRateResponse {
	return ShippingRateResponse{
		ID: r.ID, Name: r.Name, AmountCents: r.AmountCents,
		FreeOverCents: r.FreeOverCents, Active: r.Active,
	}
}

func FromShippingRates(rs []domain.ShippingRate) []ShippingRateResponse {
	out := make([]ShippingRateResponse, len(rs))
	for i := range rs {
		out[i] = FromShippingRate(&rs[i])
	}
	return out
}

// ---- coupons ----

type CreateCouponRequest struct {
	Code           string     `json:"code" binding:"required,max=64"`
	Kind           string     `json:"kind" binding:"required,oneof=PERCENT FIXED"`
	Value          int64      `json:"value" binding:"required,gt=0"`
	MinOrderCents  int64      `json:"min_order_cents" binding:"gte=0"`
	MaxRedemptions *int       `json:"max_redemptions" binding:"omitempty,gt=0"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	Active         *bool      `json:"active"`
}

type UpdateCouponRequest struct {
	Value          *int64     `json:"value" binding:"omitempty,gt=0"`
	MinOrderCents  *int64     `json:"min_order_cents" binding:"omitempty,gte=0"`
	MaxRedemptions *int       `json:"max_redemptions" binding:"omitempty,gt=0"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	Active         *bool      `json:"active"`
}

func (r UpdateCouponRequest) IsEmpty() bool {
	return r.Value == nil && r.MinOrderCents == nil && r.MaxRedemptions == nil &&
		r.StartsAt == nil && r.EndsAt == nil && r.Active == nil
}

type CouponResponse struct {
	ID             uuid.UUID  `json:"id"`
	Code           string     `json:"code"`
	Kind           string     `json:"kind"`
	Value          int64      `json:"value"`
	MinOrderCents  int64      `json:"min_order_cents"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	RedeemedCount  int        `json:"redeemed_count"`
	StartsAt       *time.Time `json:"starts_at,omitempty"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
	Active         bool       `json:"active"`
}

func FromCoupon(c *domain.Coupon) CouponResponse {
	return CouponResponse{
		ID: c.ID, Code: c.Code, Kind: string(c.Kind), Value: c.Value,
		MinOrderCents: c.MinOrderCents, MaxRedemptions: c.MaxRedemptions,
		RedeemedCount: c.RedeemedCount, StartsAt: c.StartsAt, EndsAt: c.EndsAt, Active: c.Active,
	}
}

func FromCoupons(cs []domain.Coupon) []CouponResponse {
	out := make([]CouponResponse, len(cs))
	for i := range cs {
		out[i] = FromCoupon(&cs[i])
	}
	return out
}

// CouponCheckRequest previews a code against a subtotal before checkout.
type CouponCheckRequest struct {
	Code          string `json:"code" binding:"required,max=64"`
	SubtotalCents int64  `json:"subtotal_cents" binding:"required,gt=0"`
}

type CouponCheckResponse struct {
	Valid         bool   `json:"valid"`
	DiscountCents int64  `json:"discount_cents"`
	Message       string `json:"message,omitempty"`
}
