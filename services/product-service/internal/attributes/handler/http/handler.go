// Package httphandler contains the Gin HTTP handlers for the attribute module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/product-service/internal/attributes/dto"
	"github.com/jayedbinnazir/product-service/internal/attributes/services"
	"github.com/jayedbinnazir/product-service/internal/httpx"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

// List — GET /tenants/:tenantId/categories/:categoryId/attributes
func (h *Handler) List(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	details, err := h.svc.ListForCategory(c.Request.Context(), tenantID, categoryID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetails(details))
}

// Create — POST /tenants/:tenantId/categories/:categoryId/attributes
func (h *Handler) Create(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	var req dto.CreateAttributeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.CreateAttribute(c.Request.Context(), tenantID, categoryID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromDetail(detail))
}

// Update — PATCH /tenants/:tenantId/categories/:categoryId/attributes/:attributeId
func (h *Handler) Update(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	attributeID, ok := httpx.UUIDParam(c, "attributeId")
	if !ok {
		return
	}
	var req dto.UpdateAttributeRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	detail, err := h.svc.UpdateAttribute(c.Request.Context(), tenantID, categoryID, attributeID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromDetail(detail))
}

// Delete — DELETE /tenants/:tenantId/categories/:categoryId/attributes/:attributeId
func (h *Handler) Delete(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	attributeID, ok := httpx.UUIDParam(c, "attributeId")
	if !ok {
		return
	}
	if err := h.svc.DeleteAttribute(c.Request.Context(), tenantID, categoryID, attributeID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// AddValue — POST /tenants/:tenantId/categories/:categoryId/attributes/:attributeId/values
func (h *Handler) AddValue(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	attributeID, ok := httpx.UUIDParam(c, "attributeId")
	if !ok {
		return
	}
	var req dto.AddValueRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	v, err := h.svc.AddValue(c.Request.Context(), tenantID, categoryID, attributeID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromValue(v))
}

// DeleteValue — DELETE .../attributes/:attributeId/values/:valueId
func (h *Handler) DeleteValue(c *gin.Context) {
	tenantID, categoryID, ok := h.scope(c)
	if !ok {
		return
	}
	attributeID, ok := httpx.UUIDParam(c, "attributeId")
	if !ok {
		return
	}
	valueID, ok := httpx.UUIDParam(c, "valueId")
	if !ok {
		return
	}
	if err := h.svc.DeleteValue(c.Request.Context(), tenantID, categoryID, attributeID, valueID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) scope(c *gin.Context) (tenantID, categoryID uuid.UUID, ok bool) {
	tenantID, ok = httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	categoryID, ok = httpx.UUIDParam(c, "categoryId")
	return
}
