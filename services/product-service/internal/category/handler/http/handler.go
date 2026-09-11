// Package httphandler contains the Gin HTTP handlers for the category module.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/category/dto"
	"github.com/jayedbinnazir/product-service/internal/category/services"
	"github.com/jayedbinnazir/product-service/internal/httpx"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	cats, err := h.svc.List(c.Request.Context(), tenantID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromCategories(cats))
}

func (h *Handler) Get(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	id, ok := httpx.UUIDParam(c, "categoryId")
	if !ok {
		return
	}
	cat, err := h.svc.Get(c.Request.Context(), tenantID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromCategory(cat))
}

func (h *Handler) Create(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.CreateCategoryRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	cat, err := h.svc.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromCategory(cat))
}

func (h *Handler) Update(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	id, ok := httpx.UUIDParam(c, "categoryId")
	if !ok {
		return
	}
	var req dto.UpdateCategoryRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	cat, err := h.svc.Update(c.Request.Context(), tenantID, id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromCategory(cat))
}

func (h *Handler) Delete(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	id, ok := httpx.UUIDParam(c, "categoryId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), tenantID, id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}
