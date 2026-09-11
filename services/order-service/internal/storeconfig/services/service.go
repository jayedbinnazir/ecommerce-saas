// Package services holds the store-config logic: tax/shipping/coupon CRUD plus
// the checkout money quote (shipping + discount + tax).
package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/storeconfig/domain"
	"github.com/jayedbinnazir/order-service/internal/storeconfig/dto"
)

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service { return &Service{repo: repo} }

// ---------------------------------------------------------------------
// Checkout quote — the money breakdown
// ---------------------------------------------------------------------

// Quote is the computed money breakdown for an order.
type Quote struct {
	SubtotalCents   int64
	DiscountCents   int64
	ShippingCents   int64
	TaxCents        int64
	GrandTotalCents int64
	CouponID        *uuid.UUID // set when a coupon applied; order-service redeems it after the order is written
	CouponCode      *string
}

// QuoteFor computes shipping + discount + tax for a cart worth subtotalCents.
func (s *Service) QuoteFor(ctx context.Context, tenantID uuid.UUID, subtotalCents int64, shippingRateID uuid.UUID, couponCode string) (*Quote, error) {
	rate, err := s.repo.GetShippingRate(ctx, tenantID, shippingRateID)
	if err != nil {
		return nil, err
	}
	if !rate.Active {
		return nil, domain.ErrShippingRateInactive
	}

	q := &Quote{SubtotalCents: subtotalCents}

	if code := strings.TrimSpace(couponCode); code != "" {
		coupon, err := s.repo.GetCouponByCode(ctx, tenantID, code)
		if err != nil {
			return nil, err
		}
		if err := coupon.Redeemable(subtotalCents, time.Now().UTC()); err != nil {
			return nil, err
		}
		q.DiscountCents = coupon.DiscountFor(subtotalCents)
		q.CouponID = &coupon.ID
		q.CouponCode = &coupon.Code
	}

	afterDiscount := subtotalCents - q.DiscountCents
	q.ShippingCents = rate.CostFor(afterDiscount)

	tax, err := s.repo.GetTaxConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tax.TaxRateBps > 0 {
		bps := int64(tax.TaxRateBps)
		if tax.TaxInclusive {
			net := roundDiv(afterDiscount*10000, 10000+bps)
			q.TaxCents = afterDiscount - net
			q.GrandTotalCents = afterDiscount + q.ShippingCents
		} else {
			q.TaxCents = roundDiv(afterDiscount*bps, 10000)
			q.GrandTotalCents = afterDiscount + q.ShippingCents + q.TaxCents
		}
	} else {
		q.GrandTotalCents = afterDiscount + q.ShippingCents
	}
	return q, nil
}

// RedeemCoupon bumps a coupon's redemption count (best-effort — called after the
// order is safely written).
func (s *Service) RedeemCoupon(ctx context.Context, couponID uuid.UUID) error {
	return s.repo.Redeem(ctx, couponID)
}

// CheckCoupon previews a code against a subtotal without failing hard.
func (s *Service) CheckCoupon(ctx context.Context, tenantID uuid.UUID, code string, subtotalCents int64) dto.CouponCheckResponse {
	coupon, err := s.repo.GetCouponByCode(ctx, tenantID, strings.TrimSpace(code))
	if err != nil {
		return dto.CouponCheckResponse{Valid: false, Message: "no such coupon"}
	}
	if err := coupon.Redeemable(subtotalCents, time.Now().UTC()); err != nil {
		return dto.CouponCheckResponse{Valid: false, Message: err.Error()}
	}
	return dto.CouponCheckResponse{Valid: true, DiscountCents: coupon.DiscountFor(subtotalCents)}
}

// ---------------------------------------------------------------------
// Tax config
// ---------------------------------------------------------------------

func (s *Service) GetTaxConfig(ctx context.Context, tenantID uuid.UUID) (*domain.TaxConfig, error) {
	return s.repo.GetTaxConfig(ctx, tenantID)
}

func (s *Service) SetTaxConfig(ctx context.Context, tenantID uuid.UUID, req dto.TaxConfigRequest) (*domain.TaxConfig, error) {
	if req.TaxRateBps < 0 || req.TaxRateBps > 10000 {
		return nil, domain.ErrInvalidTaxRate
	}
	c := &domain.TaxConfig{TenantID: tenantID, TaxRateBps: req.TaxRateBps, TaxInclusive: req.TaxInclusive}
	if err := s.repo.UpsertTaxConfig(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ---------------------------------------------------------------------
// Shipping rates
// ---------------------------------------------------------------------

func (s *Service) ListShippingRates(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]domain.ShippingRate, error) {
	return s.repo.ListShippingRates(ctx, tenantID, activeOnly)
}

func (s *Service) CreateShippingRate(ctx context.Context, tenantID uuid.UUID, req dto.CreateShippingRateRequest) (*domain.ShippingRate, error) {
	r := &domain.ShippingRate{
		TenantID:      tenantID,
		Name:          strings.TrimSpace(req.Name),
		AmountCents:   req.AmountCents,
		FreeOverCents: req.FreeOverCents,
		Active:        req.Active == nil || *req.Active,
	}
	if err := s.repo.CreateShippingRate(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) UpdateShippingRate(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateShippingRateRequest) (*domain.ShippingRate, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	r, err := s.repo.GetShippingRate(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		r.Name = strings.TrimSpace(*req.Name)
	}
	if req.AmountCents != nil {
		r.AmountCents = *req.AmountCents
	}
	if req.FreeOverCents != nil {
		r.FreeOverCents = req.FreeOverCents
	}
	if req.Active != nil {
		r.Active = *req.Active
	}
	if err := s.repo.UpdateShippingRate(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Service) DeleteShippingRate(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteShippingRate(ctx, tenantID, id)
}

// ---------------------------------------------------------------------
// Coupons
// ---------------------------------------------------------------------

func (s *Service) ListCoupons(ctx context.Context, tenantID uuid.UUID) ([]domain.Coupon, error) {
	return s.repo.ListCoupons(ctx, tenantID)
}

func (s *Service) CreateCoupon(ctx context.Context, tenantID uuid.UUID, req dto.CreateCouponRequest) (*domain.Coupon, error) {
	kind := domain.CouponKind(req.Kind)
	if !kind.Valid() {
		return nil, domain.ErrInvalidCouponKind
	}
	c := &domain.Coupon{
		TenantID:       tenantID,
		Code:           strings.TrimSpace(req.Code),
		Kind:           kind,
		Value:          req.Value,
		MinOrderCents:  req.MinOrderCents,
		MaxRedemptions: req.MaxRedemptions,
		StartsAt:       req.StartsAt,
		EndsAt:         req.EndsAt,
		Active:         req.Active == nil || *req.Active,
	}
	if err := s.repo.CreateCoupon(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) UpdateCoupon(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateCouponRequest) (*domain.Coupon, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	c, err := s.repo.GetCoupon(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Value != nil {
		c.Value = *req.Value
	}
	if req.MinOrderCents != nil {
		c.MinOrderCents = *req.MinOrderCents
	}
	if req.MaxRedemptions != nil {
		c.MaxRedemptions = req.MaxRedemptions
	}
	if req.StartsAt != nil {
		c.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		c.EndsAt = req.EndsAt
	}
	if req.Active != nil {
		c.Active = *req.Active
	}
	if err := s.repo.UpdateCoupon(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) DeleteCoupon(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteCoupon(ctx, tenantID, id)
}

// roundDiv is n/d rounded half-up (n, d > 0).
func roundDiv(n, d int64) int64 { return (n + d/2) / d }
