// Package routes wires the order module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/cartclient"
	"github.com/jayedbinnazir/order-service/internal/events"
	"github.com/jayedbinnazir/order-service/internal/httpx"
	"github.com/jayedbinnazir/order-service/internal/inventoryclient"
	httphandler "github.com/jayedbinnazir/order-service/internal/order/handler/http"
	"github.com/jayedbinnazir/order-service/internal/order/repository"
	"github.com/jayedbinnazir/order-service/internal/order/services"
	"github.com/jayedbinnazir/order-service/internal/paymentclient"
	storeconfig "github.com/jayedbinnazir/order-service/internal/storeconfig/services"
)

// Register mounts /tenants/:tenantId/orders[...]. Every route needs a valid
// access token. A customer sees and acts on their own orders; a tenant
// ADMIN/MANAGER can list all (?scope=all), read any, and fulfil.
func Register(rg *gin.RouterGroup, db *sql.DB, cart *cartclient.Client, inventory *inventoryclient.Client, payments *paymentclient.Client, cfg *storeconfig.Service, publisher *events.Publisher, authzClient *authz.Client, guards httpx.Guards) {
	svc := services.New(repository.New(db), cart, inventory, payments, cfg, publisher, authzClient)
	h := httphandler.New(svc)

	orders := rg.Group("/tenants/:tenantId/orders", guards.Authenticated)
	{
		orders.POST("", h.Checkout)
		orders.GET("", h.List)
		orders.GET("/:orderId", h.Get)
		orders.POST("/:orderId/pay", h.Pay)
		orders.POST("/:orderId/cancel", h.Cancel)
		orders.POST("/:orderId/fulfil", h.Fulfil)
	}
}
