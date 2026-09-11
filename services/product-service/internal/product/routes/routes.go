// Package routes wires the product module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/authz"
	"github.com/jayedbinnazir/product-service/internal/httpx"
	objstore "github.com/jayedbinnazir/product-service/internal/infrastructure/s3"
	httphandler "github.com/jayedbinnazir/product-service/internal/product/handler/http"
	"github.com/jayedbinnazir/product-service/internal/product/services"
)

// Register mounts:
//   - /tenants/:tenantId/catalog[...]     public storefront (ACTIVE products only)
//   - /tenants/:tenantId/products[...]     management, tenant ADMIN or MANAGER
func Register(rg *gin.RouterGroup, db *sql.DB, storage *objstore.Storage, guards httpx.Guards) {
	h := httphandler.New(services.New(db, storage))

	// ---- storefront (public) ----
	catalog := rg.Group("/tenants/:tenantId/catalog")
	{
		catalog.GET("", h.ListPublished)
		catalog.GET("/:productId", h.GetPublished)
	}

	// ---- management ----
	manage := rg.Group("/tenants/:tenantId/products",
		guards.Authenticated,
		guards.TenantRole("tenantId", authz.RoleAdmin, authz.RoleManager),
	)
	{
		manage.GET("", h.List)
		manage.POST("", h.Create)
		manage.GET("/:productId", h.Get)
		manage.PATCH("/:productId", h.Update)
		manage.DELETE("/:productId", h.Delete)

		manage.PUT("/:productId/categories", h.SetCategories)
		manage.PUT("/:productId/specs", h.SetSpecs)

		manage.POST("/:productId/variants", h.AddVariant)
		manage.PATCH("/:productId/variants/:variantId", h.UpdateVariant)
		manage.DELETE("/:productId/variants/:variantId", h.DeleteVariant)

		manage.POST("/:productId/images/upload-url", h.CreateImageUploadURL)
		manage.POST("/:productId/images", h.ConfirmImage)
		manage.PATCH("/:productId/images/:imageId", h.UpdateImage)
		manage.DELETE("/:productId/images/:imageId", h.DeleteImage)
	}
}
