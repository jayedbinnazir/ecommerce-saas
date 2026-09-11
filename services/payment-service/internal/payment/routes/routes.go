// Package routes wires the payment module.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/payment-service/internal/authz"
	"github.com/jayedbinnazir/payment-service/internal/events"
	"github.com/jayedbinnazir/payment-service/internal/gateway"
	"github.com/jayedbinnazir/payment-service/internal/httpx"
	"github.com/jayedbinnazir/payment-service/internal/middleware"
	httphandler "github.com/jayedbinnazir/payment-service/internal/payment/handler/http"
	"github.com/jayedbinnazir/payment-service/internal/payment/repository"
	"github.com/jayedbinnazir/payment-service/internal/payment/services"
)

// Register mounts:
//   - GET /tenants/:tenantId/payments?order_id=<uuid>       owner or manager
//   - GET /tenants/:tenantId/payments/:paymentId            owner or manager
//   - POST /webhooks/stripe                                 Stripe (signature-verified)
//   - /internal/tenants/:tenantId/payments[...]             order-service (X-Internal-Key)
func Register(rg *gin.RouterGroup, db *sql.DB, gw gateway.Gateway, publisher *events.Publisher, authzClient *authz.Client, guards httpx.Guards, internalKey string) {
	svc := services.New(repository.New(db), gw, publisher, authzClient)
	h := httphandler.New(svc, gw)

	read := rg.Group("/tenants/:tenantId", guards.Authenticated)
	{
		read.GET("/payments", h.ListPayments)          // ?order_id=<uuid>
		read.GET("/payments/:paymentId", h.GetPayment) // by payment id
	}

	rg.POST("/webhooks/stripe", h.StripeWebhook)

	internal := rg.Group("/internal/tenants/:tenantId/payments", middleware.RequireInternalKey(internalKey))
	{
		internal.POST("", h.CreatePayment)
		internal.POST("/:paymentId/sync", h.Sync)
		internal.POST("/:paymentId/settle", h.Settle)
		internal.POST("/:paymentId/refund", h.Refund)
	}
}
