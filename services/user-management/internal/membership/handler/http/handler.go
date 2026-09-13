// Package httphandler contains the Gin HTTP handlers for the membership module.
package httphandler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/membership/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/membership/services"
	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
)

type Handler struct {
	svc *services.Service
}

func New(svc *services.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListMembers(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	views, err := h.svc.ListMembers(c.Request.Context(), tenantID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromViews(views))
}

func (h *Handler) AddMember(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	var req dto.AddMemberRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	m, err := h.svc.AddMember(c.Request.Context(), tenantID, uuid.MustParse(req.UserID), roledomain.RoleName(req.Role))
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.Created(c, dto.FromMembership(m))
}

func (h *Handler) ChangeRole(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	var req dto.ChangeRoleRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	m, err := h.svc.ChangeRole(c.Request.Context(), tenantID, userID, roledomain.RoleName(req.Role))
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromMembership(m))
}

func (h *Handler) RemoveMember(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.svc.RemoveMember(c.Request.Context(), tenantID, userID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ListMemberPermissions(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	perms, err := h.svc.ListMemberPermissions(c.Request.Context(), tenantID, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, gin.H{"permissions": perms})
}

func (h *Handler) GrantMemberPermission(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	var req dto.GrantPermissionRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.svc.GrantMemberPermission(c.Request.Context(), tenantID, userID, uuid.MustParse(req.PermissionID)); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) RevokeMemberPermission(c *gin.Context) {
	tenantID, ok := httpx.UUIDParam(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	permissionID, ok := httpx.UUIDParam(c, "permissionId")
	if !ok {
		return
	}
	if err := h.svc.RevokeMemberPermission(c.Request.Context(), tenantID, userID, permissionID); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// ListMine — GET /me/memberships.
func (h *Handler) ListMine(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	views, err := h.svc.ListForUser(c.Request.Context(), principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, dto.FromViews(views))
}
