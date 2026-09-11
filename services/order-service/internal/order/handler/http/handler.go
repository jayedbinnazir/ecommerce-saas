// Package httphandler contains the Gin HTTP handlers for the order module.
package httphandler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/order/domain"
	"github.com/jayedbinnazir/order-service/internal/order/dto"
	"github.com/jayedbinnazir/order-service/internal/order/services"
	"github.com/jayedbinnazir/order-service/internal/platform"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Checkout(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	var req dto.CheckoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	// Idempotency-Key makes a retried checkout return the same order.
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))

	detail, created, err := h.svc.Checkout(c.Request.Context(), tenantID, customerID, token, key, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if created {
		httpx.Created(c, dto.FromDetail(detail))
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) List(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)

	query := services.ListQuery{
		All:  strings.EqualFold(c.Query("scope"), "all"),
		Page: platform.Page{Limit: limit, Offset: offset},
	}
	if st := c.Query("status"); st != "" {
		status := domain.Status(strings.ToUpper(st))
		if !validStatus(status) {
			_ = c.Error(httpx.BadRequest("query parameter status must be PENDING_PAYMENT, PAID, FULFILLED or CANCELLED"))
			return
		}
		query.Status = &status
	}

	details, err := h.svc.List(c.Request.Context(), tenantID, customerID, token, query)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromDetails(details), limit, offset)
}

func (h *Handler) Get(c *gin.Context) {
	tenantID, customerID, token, orderID, ok := h.orderScope(c)
	if !ok {
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), tenantID, customerID, token, orderID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) Pay(c *gin.Context) {
	tenantID, customerID, token, orderID, ok := h.orderScope(c)
	if !ok {
		return
	}
	var req dto.PayRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.Pay(c.Request.Context(), tenantID, customerID, token, orderID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPayResult(result.Detail, result.ClientSecret))
}

func (h *Handler) Cancel(c *gin.Context) {
	tenantID, customerID, token, orderID, ok := h.orderScope(c)
	if !ok {
		return
	}
	detail, err := h.svc.Cancel(c.Request.Context(), tenantID, customerID, token, orderID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) Fulfil(c *gin.Context) {
	tenantID, _, token, orderID, ok := h.orderScope(c)
	if !ok {
		return
	}
	var req dto.FulfilRequest
	_ = c.ShouldBindJSON(&req) // body optional

	detail, err := h.svc.Fulfil(c.Request.Context(), tenantID, token, orderID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// ---- helpers ----

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

func (h *Handler) orderScope(c *gin.Context) (tenantID, customerID uuid.UUID, rawToken string, orderID uuid.UUID, ok bool) {
	tenantID, customerID, rawToken, ok = h.caller(c)
	if !ok {
		return
	}
	orderID, ok = httpx.UUIDParam(c, "orderId")
	return
}

func validStatus(s domain.Status) bool {
	switch s {
	case domain.StatusPendingPayment, domain.StatusConfirmed, domain.StatusFulfilled, domain.StatusCancelled:
		return true
	default:
		return false
	}
}
