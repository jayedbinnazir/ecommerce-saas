// Package routes wires the returns module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/inventoryclient"
	orderrepo "github.com/jayedbinnazir/order-service/internal/order/repository"
	"github.com/jayedbinnazir/order-service/internal/paymentclient"
	httphandler "github.com/jayedbinnazir/order-service/internal/returns/handler/http"
	"github.com/jayedbinnazir/order-service/internal/returns/repository"
	"github.com/jayedbinnazir/order-service/internal/returns/services"
)

// Register mounts:
//   - POST /tenants/:tenantId/orders/:orderId/returns   customer — request a return
//   - GET  /tenants/:tenantId/orders/:orderId/returns   owner or manager
//   - GET  /tenants/:tenantId/returns                   manager — all (?status=)
//   - GET  /tenants/:tenantId/returns/:returnId         owner or manager
//   - POST /tenants/:tenantId/returns/:returnId/resolve manager — approve / reject
func Register(rg *gin.RouterGroup, db *sql.DB, payments *paymentclient.Client, inventory *inventoryclient.Client, authzClient *authz.Client, guards httpx.Guards) {
	svc := services.New(repository.New(db), orderrepo.New(db), payments, inventory, authzClient)
	h := httphandler.New(svc)

	onOrder := rg.Group("/tenants/:tenantId/orders/:orderId/returns", guards.Authenticated)
	{
		onOrder.POST("", h.Request)
		onOrder.GET("", h.ListForOrder)
	}

	returns := rg.Group("/tenants/:tenantId/returns", guards.Authenticated)
	{
		returns.GET("", h.List)
		returns.GET("/:returnId", h.Get)
		returns.POST("/:returnId/resolve", h.Resolve)
	}
}
