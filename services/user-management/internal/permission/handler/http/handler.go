// Package httphandler contains the Gin HTTP handlers for the permission module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/permission/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/permission/services"
)

type PermissionHandler struct {
	perms  *services.PermissionService
	grants *services.GrantService
}

func NewPermissionHandler(perms *services.PermissionService, grants *services.GrantService) *PermissionHandler {
	return &PermissionHandler{perms: perms, grants: grants}
}

// ---- catalog ----

func (h *PermissionHandler) Create(c *gin.Context) {
	var req dto.CreatePermissionRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.perms.Create(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromPermission(p))
}

func (h *PermissionHandler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.perms.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPermission(p))
}

func (h *PermissionHandler) List(c *gin.Context) {
	limit, offset := httpx.Pagination(c)
	ps, err := h.perms.List(c.Request.Context(), limit, offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromPermissions(ps), limit, offset)
}

func (h *PermissionHandler) Update(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdatePermissionRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.perms.Update(c.Request.Context(), id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPermission(p))
}

func (h *PermissionHandler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.perms.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ---- role grants ----

func (h *PermissionHandler) ListRoleGrants(c *gin.Context) {
	roleID, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	ps, err := h.grants.ListByRole(c.Request.Context(), roleID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromPermissions(ps))
}

func (h *PermissionHandler) AssignRoleGrant(c *gin.Context) {
	roleID, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	var req dto.GrantRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.grants.AssignToRole(c.Request.Context(), roleID, uuid.MustParse(req.PermissionID)); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *PermissionHandler) RevokeRoleGrant(c *gin.Context) {
	roleID, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	permID, ok := httpx.UUIDParam(c, "permissionId")
	if !ok {
		return
	}
	if err := h.grants.RevokeFromRole(c.Request.Context(), roleID, permID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}
