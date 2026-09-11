// Package routes wires the review module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/httpx"
	productrepo "github.com/jayedbinnazir/product-service/internal/product/repository"
	httphandler "github.com/jayedbinnazir/product-service/internal/review/handler/http"
	"github.com/jayedbinnazir/product-service/internal/review/repository"
	"github.com/jayedbinnazir/product-service/internal/review/services"
)

// Register mounts, under the public storefront path:
//   - GET    /tenants/:tenantId/catalog/:productId/reviews          public
//   - POST   /tenants/:tenantId/catalog/:productId/reviews          any signed-in shopper (one per product)
//   - DELETE /tenants/:tenantId/catalog/:productId/reviews/:reviewId  the review's author
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards) {
	svc := services.New(repository.New(db), productrepo.NewProductRepository(db))
	h := httphandler.New(svc)

	base := rg.Group("/tenants/:tenantId/catalog/:productId/reviews")
	{
		base.GET("", h.List)

		auth := base.Group("", guards.Authenticated)
		{
			auth.POST("", h.Create)
			auth.DELETE("/:reviewId", h.Delete)
		}
	}
}
