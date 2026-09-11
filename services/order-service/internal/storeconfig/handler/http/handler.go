// Package httphandler contains the Gin HTTP handlers for the store-config module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/storeconfig/dto"
	"github.com/jayedbinnazir/order-service/internal/storeconfig/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// ---- tax ----

func (h *Handler) GetTaxConfig(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	cfg, err := h.svc.GetTaxConfig(c.Request.Context(), tenantID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromTaxConfig(cfg))
}

func (h *Handler) SetTaxConfig(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.TaxConfigRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	cfg, err := h.svc.SetTaxConfig(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromTaxConfig(cfg))
}

// ---- shipping rates ----

// List returns every shipping rate for the tenant (checkout shows the active ones).
func (h *Handler) ListShippingRates(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	rates, err := h.svc.ListShippingRates(c.Request.Context(), tenantID, false)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromShippingRates(rates))
}

func (h *Handler) CreateShippingRate(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreateShippingRateRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	rate, err := h.svc.CreateShippingRate(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromShippingRate(rate))
}

func (h *Handler) UpdateShippingRate(c *gin.Context) {
	tenantID, id, ok := h.scope(c, "rateId")
	if !ok {
		return
	}
	var req dto.UpdateShippingRateRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	rate, err := h.svc.UpdateShippingRate(c.Request.Context(), tenantID, id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromShippingRate(rate))
}

func (h *Handler) DeleteShippingRate(c *gin.Context) {
	tenantID, id, ok := h.scope(c, "rateId")
	if !ok {
		return
	}
	if err := h.svc.DeleteShippingRate(c.Request.Context(), tenantID, id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ---- coupons ----

func (h *Handler) ListCoupons(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	coupons, err := h.svc.ListCoupons(c.Request.Context(), tenantID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromCoupons(coupons))
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreateCouponRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	coupon, err := h.svc.CreateCoupon(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromCoupon(coupon))
}

func (h *Handler) UpdateCoupon(c *gin.Context) {
	tenantID, id, ok := h.scope(c, "couponId")
	if !ok {
		return
	}
	var req dto.UpdateCouponRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	coupon, err := h.svc.UpdateCoupon(c.Request.Context(), tenantID, id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromCoupon(coupon))
}

func (h *Handler) DeleteCoupon(c *gin.Context) {
	tenantID, id, ok := h.scope(c, "couponId")
	if !ok {
		return
	}
	if err := h.svc.DeleteCoupon(c.Request.Context(), tenantID, id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// CheckCoupon previews a code against a subtotal before checkout.
func (h *Handler) CheckCoupon(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CouponCheckRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	httpx.OK(c, h.svc.CheckCoupon(c.Request.Context(), tenantID, req.Code, req.SubtotalCents))
}

func (h *Handler) scope(c *gin.Context, idParam string) (tenantID, id uuid.UUID, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	id, ok = httpx.UUIDParam(c, idParam)
	return
}
