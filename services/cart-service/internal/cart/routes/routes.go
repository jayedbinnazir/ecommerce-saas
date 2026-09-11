// Package routes wires the cart module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/cart-service/internal/availability"
	httphandler "github.com/jayedbinnazir/cart-service/internal/cart/handler/http"
	"github.com/jayedbinnazir/cart-service/internal/cart/services"
	"github.com/jayedbinnazir/cart-service/internal/catalog"
	"github.com/jayedbinnazir/cart-service/internal/httpx"
)

// Register mounts /tenants/:tenantId/cart[...]. Every route needs a valid access
// token; the cart belongs to whoever the token identifies.
func Register(rg *gin.RouterGroup, db *sql.DB, catalogClient *catalog.Client, inventoryClient *availability.Client, guards httpx.Guards) {
	h := httphandler.New(services.New(db, catalogClient, inventoryClient))

	cart := rg.Group("/tenants/:tenantId/cart", guards.Authenticated)
	{
		cart.GET("", h.Get)
		cart.DELETE("", h.Clear)

		cart.POST("/items", h.AddItem)
		cart.PATCH("/items/:sku", h.SetItemQuantity)
		cart.DELETE("/items/:sku", h.RemoveItem)
	}
}
