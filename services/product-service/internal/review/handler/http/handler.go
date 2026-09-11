// Package httphandler contains the Gin HTTP handlers for the review module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/httpx"
	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/review/dto"
	"github.com/jayedbinnazir/product-service/internal/review/services"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// List — GET /tenants/:tenantId/catalog/:productId/reviews  (public)
func (h *Handler) List(c *gin.Context) {
	_, productID, ok := h.scope(c)
	if !ok {
		return
	}
	limit, offset := httpx.Pagination(c)

	reviews, err := h.svc.List(c.Request.Context(), productID, platform.Page{Limit: limit, Offset: offset})
	if err != nil {
		_ = c.Error(err)
		return
	}
	summary, err := h.svc.Summary(c.Request.Context(), productID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, gin.H{
		"summary": dto.FromSummary(summary),
		"items":   dto.FromReviews(reviews),
		"limit":   limit,
		"offset":  offset,
	})
}

// Create — POST /tenants/:tenantId/catalog/:productId/reviews  (authenticated)
func (h *Handler) Create(c *gin.Context) {
	tenantID, productID, ok := h.scope(c)
	if !ok {
		return
	}
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	var req dto.CreateReviewRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	review, err := h.svc.Create(c.Request.Context(), tenantID, productID, principal.UserID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromReview(review))
}

// Delete — DELETE /tenants/:tenantId/catalog/:productId/reviews/:reviewId  (author)
func (h *Handler) Delete(c *gin.Context) {
	tenantID, _, ok := h.scope(c)
	if !ok {
		return
	}
	reviewID, ok := httpx.UUIDParam(c, "reviewId")
	if !ok {
		return
	}
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), tenantID, reviewID, principal.UserID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) scope(c *gin.Context) (tenantID, productID uuid.UUID, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	productID, ok = httpx.UUIDParam(c, "productId")
	return
}
