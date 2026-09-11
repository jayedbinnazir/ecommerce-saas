// Package routes wires the store-config module (tax / shipping / coupons).
package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/httpx"
	httphandler "github.com/jayedbinnazir/order-service/internal/storeconfig/handler/http"
	"github.com/jayedbinnazir/order-service/internal/storeconfig/services"
)

// Register mounts the tenant checkout-config APIs. Reads that a shopper needs at
// checkout (shipping rates, coupon check) are open to any authenticated user;
// everything else is tenant ADMIN/MANAGER.
func Register(rg *gin.RouterGroup, svc *services.Service, guards httpx.Guards) {
	h := httphandler.New(svc)

	manager := func(rg *gin.RouterGroup) *gin.RouterGroup {
		return rg.Group("", guards.Authenticated, guards.TenantRole("tenantId", authz.RoleAdmin, authz.RoleManager))
	}

	// ---- tax config (manager) ----
	tax := rg.Group("/tenants/:tenantId/tax-config")
	m := manager(tax)
	m.GET("", h.GetTaxConfig)
	m.PUT("", h.SetTaxConfig)

	// ---- shipping rates ----
	ship := rg.Group("/tenants/:tenantId/shipping-rates")
	ship.GET("", guards.Authenticated, h.ListShippingRates)
	sm := manager(ship)
	sm.POST("", h.CreateShippingRate)
	sm.PATCH("/:rateId", h.UpdateShippingRate)
	sm.DELETE("/:rateId", h.DeleteShippingRate)

	// ---- coupons ----
	cpn := rg.Group("/tenants/:tenantId/coupons")
	cm := manager(cpn)
	cm.GET("", h.ListCoupons)
	cm.POST("", h.CreateCoupon)
	cm.PATCH("/:couponId", h.UpdateCoupon)
	cm.DELETE("/:couponId", h.DeleteCoupon)

	// ---- coupon preview (any authenticated shopper) ----
	rg.POST("/tenants/:tenantId/coupon-check", guards.Authenticated, h.CheckCoupon)
}
