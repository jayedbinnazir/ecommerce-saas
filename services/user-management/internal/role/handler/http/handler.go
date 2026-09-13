// Package httphandler contains the Gin HTTP handlers for the role module.
package httphandler

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/services"
)

type RoleHandler struct {
	svc *services.RoleService
}

func NewRoleHandler(svc *services.RoleService) *RoleHandler { return &RoleHandler{svc: svc} }

func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	role, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromRole(role))
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	role, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromRole(role))
}

func (h *RoleHandler) List(c *gin.Context) {
	limit, offset := httpx.Pagination(c)
	roles, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.List(c, dto.FromRoles(roles), limit, offset)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	var req dto.UpdateRoleRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	role, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromRole(role))
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "roleId")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}
