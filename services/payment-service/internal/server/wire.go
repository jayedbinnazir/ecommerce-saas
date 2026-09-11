package server

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/payment-service/internal/auth/token"
	"github.com/jayedbinnazir/payment-service/internal/authz"
	billingroutes "github.com/jayedbinnazir/payment-service/internal/billing/routes"
	"github.com/jayedbinnazir/payment-service/internal/gateway"
	"github.com/jayedbinnazir/payment-service/internal/middleware"
	paymentroutes "github.com/jayedbinnazir/payment-service/internal/payment/routes"
)

// registerModules builds the auth guard, the Stripe gateway and mounts every
// feature module.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	gw := gateway.New(s.cfg.Stripe.SecretKey, s.cfg.Stripe.WebhookSecret)

	billingroutes.Register(api, s.db, guards, s.cfg.InternalKey)
	paymentroutes.Register(api, s.db, gw, s.events, authzClient, guards, s.cfg.InternalKey)

	return nil
}
