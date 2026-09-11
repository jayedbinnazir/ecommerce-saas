// Package routes wires the billing module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	httphandler "github.com/jayedbinnazir/payment-service/internal/billing/handler/http"
	"github.com/jayedbinnazir/payment-service/internal/billing/repository"
	"github.com/jayedbinnazir/payment-service/internal/billing/services"
	"github.com/jayedbinnazir/payment-service/internal/httpx"
	"github.com/jayedbinnazir/payment-service/internal/middleware"
)

// Register mounts:
//   - GET  /plans                              public
//   - /subscription  (GET/POST/DELETE)         the caller's own, authenticated
//   - GET  /internal/users/:userId/subscription  service-to-service (X-Internal-Key)
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards, internalKey string) {
	svc := services.New(repository.NewPlanRepository(db), repository.NewSubscriptionRepository(db))
	h := httphandler.New(svc)

	rg.GET("/plans", h.ListPlans)

	sub := rg.Group("/subscription", guards.Authenticated)
	{
		sub.GET("", h.GetMySubscription)
		sub.POST("", h.Subscribe)
		sub.DELETE("", h.Cancel)
	}

	internal := rg.Group("/internal/users/:userId", middleware.RequireInternalKey(internalKey))
	internal.GET("/subscription", h.InternalStatus)
}
