// Package httphandler contains the Gin HTTP handlers for the returns module.
package httphandler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/platform"
	"github.com/jayedbinnazir/order-service/internal/returns/domain"
	"github.com/jayedbinnazir/order-service/internal/returns/dto"
	"github.com/jayedbinnazir/order-service/internal/returns/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// Request — POST /tenants/:tenantId/orders/:orderId/returns
func (h *Handler) Request(c *gin.Context) {
	tenantID, customerID, _, ok := h.caller(c)
	if !ok {
		return
	}
	orderID, ok := httpx.UUIDParam(c, "orderId")
	if !ok {
		return
	}
	var req dto.RequestReturnRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.Request(c.Request.Context(), tenantID, customerID, orderID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromDetail(detail))
}

// ListForOrder — GET /tenants/:tenantId/orders/:orderId/returns
func (h *Handler) ListForOrder(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	orderID, ok := httpx.UUIDParam(c, "orderId")
	if !ok {
		return
	}
	details, err := h.svc.ListForOrder(c.Request.Context(), tenantID, customerID, token, orderID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetails(details))
}

// List — GET /tenants/:tenantId/returns  (manager)
func (h *Handler) List(c *gin.Context) {
	tenantID, _, token, ok := h.caller(c)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)

	var status *domain.Status
	if st := c.Query("status"); st != "" {
		s := domain.Status(strings.ToUpper(st))
		status = &s
	}
	details, err := h.svc.List(c.Request.Context(), tenantID, token, status, platform.Page{Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromDetails(details), limit, offset)
}

// Get — GET /tenants/:tenantId/returns/:returnId
func (h *Handler) Get(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	returnID, ok := httpx.UUIDParam(c, "returnId")
	if !ok {
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), tenantID, customerID, token, returnID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// Resolve — POST /tenants/:tenantId/returns/:returnId/resolve  (manager)
func (h *Handler) Resolve(c *gin.Context) {
	tenantID, _, token, ok := h.caller(c)
	if !ok {
		return
	}
	returnID, ok := httpx.UUIDParam(c, "returnId")
	if !ok {
		return
	}
	var req dto.ResolveReturnRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.Resolve(c.Request.Context(), tenantID, token, returnID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
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
