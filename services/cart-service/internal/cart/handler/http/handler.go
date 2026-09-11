// Package httphandler contains the Gin HTTP handlers for the cart module.
package httphandler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/cart-service/internal/cart/dto"
	"github.com/jayedbinnazir/cart-service/internal/cart/services"
	"github.com/jayedbinnazir/cart-service/internal/httpx"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Get(c *gin.Context) {
	tenantID, customerID, _, ok := h.caller(c)
	if !ok {
		return
	}
	detail, err := h.svc.GetCart(c.Request.Context(), tenantID, customerID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) AddItem(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	var req dto.AddItemRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.AddItem(c.Request.Context(), tenantID, customerID, token, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromDetail(detail))
}

func (h *Handler) SetItemQuantity(c *gin.Context) {
	tenantID, customerID, token, ok := h.caller(c)
	if !ok {
		return
	}
	sku := strings.TrimSpace(c.Param("sku"))
	if sku == "" {
		_ = c.Error(httpx.BadRequest("path parameter sku is required"))
		return
	}
	var req dto.UpdateItemRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.SetItemQuantity(c.Request.Context(), tenantID, customerID, token, sku, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) RemoveItem(c *gin.Context) {
	tenantID, customerID, _, ok := h.caller(c)
	if !ok {
		return
	}
	sku := strings.TrimSpace(c.Param("sku"))
	if sku == "" {
		_ = c.Error(httpx.BadRequest("path parameter sku is required"))
		return
	}
	detail, err := h.svc.RemoveItem(c.Request.Context(), tenantID, customerID, sku)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

func (h *Handler) Clear(c *gin.Context) {
	tenantID, customerID, _, ok := h.caller(c)
	if !ok {
		return
	}
	detail, err := h.svc.Clear(c.Request.Context(), tenantID, customerID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// caller reads :tenantId from the path and the shopper (id + raw token) from the
// authenticated principal.
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
