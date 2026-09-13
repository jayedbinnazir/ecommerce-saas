// Package routes wires the tenant module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/tenant/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/repository"
	"github.com/jayedbinnazir/golang-saas.git/internal/tenant/services"
)

// Build constructs the tenant service. subscription (billing) and enroller
// (membership) are injected by the composition root.
func Build(db *sql.DB, subscription domain.SubscriptionGuard, enroller domain.OwnerEnroller) *services.Service {
	return services.New(db, repository.New(db), subscription, enroller)
}

// Register mounts the store APIs. Every route needs an authenticated session;
// /admin/tenants additionally needs super-admin.
func Register(rg *gin.RouterGroup, svc *services.Service, guards httpx.Guards) {
	h := httphandler.NewTenantHandler(svc)

	tenants := rg.Group("/tenants", guards.Authenticated)
	{
		tenants.POST("", h.Create)
		tenants.GET("", h.ListMine)
		tenants.GET("/:tenantId", h.Get)
		tenants.PATCH("/:tenantId", h.Update)
		tenants.DELETE("/:tenantId", h.Delete)
	}

	rg.GET("/admin/tenants", guards.Authenticated, guards.SuperAdmin, h.ListAll)
}
