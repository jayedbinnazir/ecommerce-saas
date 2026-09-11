// Package httphandler contains the Gin HTTP handlers for the payment module.
package httphandler

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/payment-service/internal/gateway"
	"github.com/jayedbinnazir/payment-service/internal/httpx"
	"github.com/jayedbinnazir/payment-service/internal/payment/domain"
	"github.com/jayedbinnazir/payment-service/internal/payment/dto"
	"github.com/jayedbinnazir/payment-service/internal/payment/services"
)

type Handler struct {
	svc *services.Service
	gw  gateway.Gateway
}

func New(svc *services.Service, gw gateway.Gateway) *Handler {
	return &Handler{svc: svc, gw: gw}
}

// ---------------------------------------------------------------------
// Internal (order-service, X-Internal-Key)
// ---------------------------------------------------------------------

func (h *Handler) CreatePayment(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreatePaymentRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.svc.CreateForOrder(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromPayment(p, true)) // include client_secret once
}

func (h *Handler) Sync(c *gin.Context)   { h.internalTransition(c, h.svc.Sync) }
func (h *Handler) Settle(c *gin.Context) { h.internalTransition(c, h.svc.Settle) }

// Refund — POST /internal/tenants/:tenantId/payments/:paymentId/refund
// Optional body { "amount_cents": N } for a partial refund; omit for a full one.
func (h *Handler) Refund(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	paymentID, ok := httpx.UUIDParam(c, "paymentId")
	if !ok {
		return
	}
	var body struct {
		AmountCents *int64 `json:"amount_cents"`
	}
	_ = c.ShouldBindJSON(&body)

	p, err := h.svc.Refund(c.Request.Context(), tenantID, paymentID, body.AmountCents)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPayment(p, false))
}

type transitionFunc func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Payment, error)

func (h *Handler) internalTransition(c *gin.Context, fn transitionFunc) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	paymentID, ok := httpx.UUIDParam(c, "paymentId")
	if !ok {
		return
	}
	p, err := fn(c.Request.Context(), tenantID, paymentID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPayment(p, false))
}

// ---------------------------------------------------------------------
// Stripe webhook (public, signature-verified)
// ---------------------------------------------------------------------

func (h *Handler) StripeWebhook(c *gin.Context) {
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		_ = c.Error(httpx.BadRequest("could not read webhook body"))
		return
	}
	eventType, intentID, err := h.gw.VerifyWebhook(payload, c.GetHeader("Stripe-Signature"))
	if err != nil {
		_ = c.Error(httpx.Unauthorized("invalid Stripe signature"))
		return
	}
	if err := h.svc.HandleWebhook(c.Request.Context(), eventType, intentID); err != nil {
		log.Printf("[payment] webhook %s for %s: %v", eventType, intentID, err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// ---------------------------------------------------------------------
// Customer reads (Authenticated)
// ---------------------------------------------------------------------

// ListPayments currently supports the one lookup order-service and the frontend
// need: ?order_id=<uuid> returns the payment for that order.
func (h *Handler) ListPayments(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	orderID, err := uuid.Parse(c.Query("order_id"))
	if err != nil {
		_ = c.Error(httpx.BadRequest("query parameter order_id must be a UUID"))
		return
	}
	p, err := h.svc.GetForOrder(c.Request.Context(), tenantID, customerID, token, orderID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, []dto.PaymentResponse{dto.FromPayment(p, false)})
}

func (h *Handler) GetPayment(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	paymentID, ok := httpx.UUIDParam(c, "paymentId")
	if !ok {
		return
	}
	p, err := h.svc.Get(c.Request.Context(), tenantID, customerID, token, paymentID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPayment(p, false))
}

func (h *Handler) caller(c *gin.Context) (tenantID, customerID uuid.UUID, rawToken string, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	principal, ok := httpx.Principal(c)
	if !ok {
		return uuid.Nil, uuid.Nil, "", false
	}
	return tenantID, principal.UserID, principal.RawToken, true
}
