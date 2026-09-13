// Package routes wires the role module and registers it on a Gin router group.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/role/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/repository"
	"github.com/jayedbinnazir/golang-saas.git/internal/role/services"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
)

// Register mounts the role catalog API. Roles are a platform-level resource, so
// every route requires a super-admin session.
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards) {
	h := httphandler.NewRoleHandler(services.NewRoleService(repository.NewRoleRepository(db)))

	roles := rg.Group("/roles", guards.Authenticated, guards.SuperAdmin)
	{
		roles.POST("", h.Create)
		roles.GET("", h.List)
		roles.GET("/:roleId", h.Get)
		roles.PATCH("/:roleId", h.Update)
		roles.DELETE("/:roleId", h.Delete)
	}
}
