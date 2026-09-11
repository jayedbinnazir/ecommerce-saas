// Package httphandler contains the Gin HTTP handlers for the notification module.
package httphandler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/notification-service/internal/httpx"
	"github.com/jayedbinnazir/notification-service/internal/notification/dto"
	"github.com/jayedbinnazir/notification-service/internal/notification/services"
	"github.com/jayedbinnazir/notification-service/internal/platform"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// ---- internal (X-Internal-Key) ----

func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	n, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromNotification(n))
}

// ---- customer (Authenticated) ----

func (h *Handler) List(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)
	unread := strings.EqualFold(c.Query("unread"), "true")

	ns, err := h.svc.List(c.Request.Context(), userID, unread, platform.Page{Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromNotifications(ns), limit, offset)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	n, err := h.svc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, gin.H{"unread": n})
}

// MarkRead — POST /notifications/read  { "ids": [...] } or { "all": true }
func (h *Handler) MarkRead(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req dto.MarkReadRequest
	if !httpx.BindJSON(c, &req) {
		return
	}

	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, raw := range req.IDs {
		ids = append(ids, uuid.MustParse(raw))
	}

	n, err := h.svc.MarkRead(c.Request.Context(), userID, ids, req.All)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, gin.H{"marked_read": n})
}

func (h *Handler) userID(c *gin.Context) (uuid.UUID, bool) {
	p, ok := httpx.Principal(c)
	if !ok {
		return uuid.Nil, false
	}
	return p.UserID, true
}
