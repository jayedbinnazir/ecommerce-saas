// Package routes wires the attribute module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	httphandler "github.com/jayedbinnazir/product-service/internal/attributes/handler/http"
	"github.com/jayedbinnazir/product-service/internal/attributes/repository"
	"github.com/jayedbinnazir/product-service/internal/attributes/services"
	"github.com/jayedbinnazir/product-service/internal/authz"
	categoryrepo "github.com/jayedbinnazir/product-service/internal/category/repository"
	"github.com/jayedbinnazir/product-service/internal/httpx"
)

// Register mounts, all under tenant ADMIN or MANAGER:
//
//	GET/POST   /tenants/:tenantId/categories/:categoryId/attributes
//	PATCH/DEL  /tenants/:tenantId/categories/:categoryId/attributes/:attributeId
//	POST       .../attributes/:attributeId/values
//	DELETE     .../attributes/:attributeId/values/:valueId
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards) {
	svc := services.New(repository.New(db), categoryrepo.New(db))
	h := httphandler.New(svc)

	base := rg.Group("/tenants/:tenantId/categories/:categoryId/attributes",
		guards.Authenticated,
		guards.TenantRole("tenantId", authz.RoleAdmin, authz.RoleManager),
	)
	{
		base.GET("", h.List)
		base.POST("", h.Create)
		base.PATCH("/:attributeId", h.Update)
		base.DELETE("/:attributeId", h.Delete)

		base.POST("/:attributeId/values", h.AddValue)
		base.DELETE("/:attributeId/values/:valueId", h.DeleteValue)
	}
}
