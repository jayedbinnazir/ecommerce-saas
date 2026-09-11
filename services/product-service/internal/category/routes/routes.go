// Package routes wires the category module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/authz"
	httphandler "github.com/jayedbinnazir/product-service/internal/category/handler/http"
	"github.com/jayedbinnazir/product-service/internal/category/repository"
	"github.com/jayedbinnazir/product-service/internal/category/services"
	"github.com/jayedbinnazir/product-service/internal/httpx"
)

// Register mounts:
//   - GET  /tenants/:tenantId/categories[/:categoryId]  public
//   - write routes                                      tenant ADMIN or MANAGER
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards) {
	h := httphandler.New(services.New(repository.New(db)))

	base := rg.Group("/tenants/:tenantId/categories")
	{
		base.GET("", h.List)
		base.GET("/:categoryId", h.Get)

		write := base.Group("", guards.Authenticated, guards.TenantRole("tenantId", authz.RoleAdmin, authz.RoleManager))
		{
			write.POST("", h.Create)
			write.PATCH("/:categoryId", h.Update)
			write.DELETE("/:categoryId", h.Delete)
		}
	}
}
