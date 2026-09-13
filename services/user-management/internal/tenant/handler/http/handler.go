// Package httphandler contains the Gin HTTP handlers for the tenant module.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/services"
)

type TenantHandler struct {
	svc *services.Service
}

func NewTenantHandler(svc *services.Service) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// Create — POST /tenants ("register your store"). Owner is the caller.
func (h *TenantHandler) Create(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	var req dto.CreateTenantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tenant, err := h.svc.Create(c.Request.Context(), principal.UserID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromTenant(tenant))
}

// ListMine — GET /tenants (stores the caller owns).
func (h *TenantHandler) ListMine(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	tenants, err := h.svc.ListMine(c.Request.Context(), principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromTenants(tenants))
}

// ListAll — GET /admin/tenants (super-admin only, guarded at the route).
func (h *TenantHandler) ListAll(c *gin.Context) {
	limit, offset := httpx.Pagination(c)
	tenants, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromTenants(tenants), limit, offset)
}

func (h *TenantHandler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	tenant, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if _, ok := h.authorize(c, tenant); !ok {
		return
	}
	httpx.OK(c, dto.FromTenant(tenant))
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	tenant, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if _, ok := h.authorize(c, tenant); !ok {
		return
	}
	var req dto.UpdateTenantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	updated, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromTenant(updated))
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	tenant, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if _, ok := h.authorize(c, tenant); !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// authorize allows the tenant owner or any super-admin.
func (h *TenantHandler) authorize(c *gin.Context, tenant *domain.Tenant) (any, bool) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return nil, false
	}
	if principal.IsSuperAdmin || tenant.OwnerUserID == principal.UserID {
		return nil, true
	}
	_ = c.Error(httpx.Forbidden("not allowed to manage this tenant"))
	return nil, false
}
