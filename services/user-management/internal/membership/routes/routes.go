// Package routes wires the membership module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/membership/guard"
	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/membership/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/membership/services"
	permservices "github.com/jayedbinnazir/golang-saas.git/internal/permission/services"
	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
	tenantdomain "github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
)

// Build constructs the membership service + guard. The returned service also
// implements tenant/domain.OwnerEnroller for the tenant-creation flow.
func Build(db *sql.DB, tenants tenantdomain.Repository, grants *permservices.GrantService) (*services.Service, *guard.Guard) {
	svc := services.New(db, tenants, grants)
	return svc, guard.New(svc)
}

// Register mounts tenant-member management under /tenants/:tenantId/members and
// the caller's own memberships under /me/memberships.
func Register(rg *gin.RouterGroup, svc *services.Service, g *guard.Guard, guards httpx.Guards) {
	h := httphandler.New(svc)

	members := rg.Group("/tenants/:tenantId/members", guards.Authenticated)
	{
		members.GET("", g.RequireTenantRole("tenantId", roledomain.Admin, roledomain.Manager), h.ListMembers)
		members.POST("", g.RequireTenantRole("tenantId", roledomain.Admin), h.AddMember)
		members.PATCH("/:userId", g.RequireTenantRole("tenantId", roledomain.Admin), h.ChangeRole)
		members.DELETE("/:userId", g.RequireTenantRole("tenantId", roledomain.Admin), h.RemoveMember)

		perms := members.Group("/:userId/permissions", g.RequireTenantRole("tenantId", roledomain.Admin))
		{
			perms.GET("", h.ListMemberPermissions)
			perms.POST("", h.GrantMemberPermission)
			perms.DELETE("/:permissionId", h.RevokeMemberPermission)
		}
	}

	rg.GET("/me/memberships", guards.Authenticated, h.ListMine)
}
