// Package routes wires the permission module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/permission/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/permission/repository"
	"github.com/jayedbinnazir/golang-saas.git/internal/permission/services"
)

// Build constructs the permission-module services from a DB handle. Other modules
// (e.g. membership) reuse the returned GrantService for member-level grants and
// authorization checks.
func Build(db *sql.DB) (*services.PermissionService, *services.GrantService) {
	return services.NewPermissionService(repository.NewPermissionRepository(db)),
		services.NewGrantService(repository.NewGrantRepository(db))
}

// Register mounts the permission catalog and role-grant APIs (super-admin only).
func Register(rg *gin.RouterGroup, permSvc *services.PermissionService, grantSvc *services.GrantService, guards httpx.Guards) {
	h := httphandler.NewPermissionHandler(permSvc, grantSvc)

	admin := rg.Group("", guards.Authenticated, guards.SuperAdmin)

	perms := admin.Group("/permissions")
	{
		perms.POST("", h.Create)
		perms.GET("", h.List)
		perms.GET("/:id", h.Get)
		perms.PATCH("/:id", h.Update)
		perms.DELETE("/:id", h.Delete)
	}

	roleGrants := admin.Group("/roles/:roleId/permissions")
	{
		roleGrants.GET("", h.ListRoleGrants)
		roleGrants.POST("", h.AssignRoleGrant)
		roleGrants.DELETE("/:permissionId", h.RevokeRoleGrant)
	}
}
