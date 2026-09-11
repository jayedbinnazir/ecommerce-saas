// Package httphandler contains the Gin HTTP handlers for the billing module.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/payment-service/internal/billing/dto"
	"github.com/jayedbinnazir/payment-service/internal/billing/services"
	"github.com/jayedbinnazir/payment-service/internal/httpx"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.svc.ListPlans(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPlans(plans))
}

func (h *Handler) GetMySubscription(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	sub, err := h.svc.GetMySubscription(c.Request.Context(), principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromSubscription(sub))
}

func (h *Handler) Subscribe(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	var req dto.SubscribeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	sub, err := h.svc.Subscribe(c.Request.Context(), principal.UserID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromSubscription(sub))
}

func (h *Handler) Cancel(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	sub, err := h.svc.Cancel(c.Request.Context(), principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromSubscription(sub))
}

// InternalStatus is the entitlement check user-management calls (X-Internal-Key)
// before letting a user create a store.
func (h *Handler) InternalStatus(c *gin.Context) {
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	status, err := h.svc.Status(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, status)
}
