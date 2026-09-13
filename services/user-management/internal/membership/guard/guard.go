// Package guard provides tenant-scoped RBAC middleware built on membership data.
package guard

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/membership/services"
	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
)

type Guard struct {
	svc *services.Service
}

func New(svc *services.Service) *Guard { return &Guard{svc: svc} }

// RequireTenantRole allows super-admins unconditionally, otherwise requires the
// caller to hold one of the listed roles in the tenant named by the given path
// parameter. Must run after the Authenticated guard.
func (g *Guard) RequireTenantRole(tenantParam string, allowed ...roledomain.RoleName) gin.HandlerFunc {
	allow := make(map[roledomain.RoleName]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}

	return func(c *gin.Context) {
		principal, ok := httpx.Principal(c)
		if !ok {
			c.Abort()
			return
		}
		if principal.IsSuperAdmin {
			c.Next()
			return
		}

		tenantID, ok := httpx.UUIDParam(c, tenantParam)
		if !ok {
			c.Abort()
			return
		}

		view, err := g.svc.Resolve(c.Request.Context(), tenantID, principal.UserID)
		if err != nil {
			_ = c.Error(httpx.Forbidden("you are not a member of this tenant"))
			c.Abort()
			return
		}
		if _, ok := allow[view.RoleName]; !ok {
			_ = c.Error(httpx.Forbidden("insufficient role for this action"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission allows super-admins and tenant ADMINs unconditionally,
// otherwise requires the caller's effective permission set to contain perm.
func (g *Guard) RequirePermission(tenantParam, perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := httpx.Principal(c)
		if !ok {
			c.Abort()
			return
		}
		if principal.IsSuperAdmin {
			c.Next()
			return
		}

		tenantID, ok := httpx.UUIDParam(c, tenantParam)
		if !ok {
			c.Abort()
			return
		}

		view, err := g.svc.Resolve(c.Request.Context(), tenantID, principal.UserID)
		if err != nil {
			_ = c.Error(httpx.Forbidden("you are not a member of this tenant"))
			c.Abort()
			return
		}
		if view.RoleName == roledomain.Admin {
			c.Next()
			return
		}

		has, err := g.svc.HasMembershipPermission(c.Request.Context(), view.ID, perm)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		if !has {
			_ = c.Error(httpx.Forbidden("missing permission: " + perm))
			c.Abort()
			return
		}
		c.Next()
	}
}
