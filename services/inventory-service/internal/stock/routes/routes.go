// Package routes wires the stock module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/inventory-service/internal/authz"
	"github.com/jayedbinnazir/inventory-service/internal/httpx"
	"github.com/jayedbinnazir/inventory-service/internal/middleware"
	httphandler "github.com/jayedbinnazir/inventory-service/internal/stock/handler/http"
	"github.com/jayedbinnazir/inventory-service/internal/stock/services"
)

// Register mounts:
//   - GET /tenants/:tenantId/stock/:sku            public — availability only
//   - /tenants/:tenantId/inventory[...]            management, tenant ADMIN or MANAGER
//   - /internal/tenants/:tenantId/stock/{reserve,release,ship}
//     service-to-service (X-Internal-Key), used by order-service at checkout
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards, internalKey string) {
	h := httphandler.New(services.New(db))

	// ---- storefront (public) ----
	rg.GET("/tenants/:tenantId/stock/:sku", h.Availability)

	// ---- management ----
	manage := rg.Group("/tenants/:tenantId/inventory",
		guards.Authenticated,
		guards.TenantRole("tenantId", authz.RoleAdmin, authz.RoleManager),
	)
	{
		manage.GET("", h.List)
		manage.POST("", h.Create)
		manage.GET("/:sku", h.Get)
		manage.PATCH("/:sku", h.Update)
		manage.DELETE("/:sku", h.Delete)

		manage.POST("/:sku/receive", h.Receive)
		manage.POST("/:sku/adjust", h.Adjust)
		manage.POST("/:sku/reserve", h.Reserve)
		manage.POST("/:sku/release", h.Release)
		manage.POST("/:sku/ship", h.Ship)

		manage.GET("/:sku/movements", h.Movements)
	}

	// ---- internal bulk operations (service-to-service) ----
	internal := rg.Group("/internal/tenants/:tenantId/stock", middleware.RequireInternalKey(internalKey))
	{
		internal.POST("/reserve", h.ReserveBulk)
		internal.POST("/release", h.ReleaseBulk)
		internal.POST("/ship", h.ShipBulk)
		internal.POST("/restock", h.RestockBulk)
	}
}
