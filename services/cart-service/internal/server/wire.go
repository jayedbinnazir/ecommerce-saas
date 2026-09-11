package server

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/cart-service/internal/auth/token"
	"github.com/jayedbinnazir/cart-service/internal/authz"
	"github.com/jayedbinnazir/cart-service/internal/availability"
	cartroutes "github.com/jayedbinnazir/cart-service/internal/cart/routes"
	"github.com/jayedbinnazir/cart-service/internal/catalog"
	"github.com/jayedbinnazir/cart-service/internal/middleware"
)

// registerModules builds the auth guard, the downstream clients and mounts every
// feature module.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	catalogClient := catalog.NewClient(s.cfg.Services.ProductServiceURL)
	inventoryClient := availability.NewClient(s.cfg.Services.InventoryServiceURL)

	cartroutes.Register(api, s.db, catalogClient, inventoryClient, guards)

	return nil
}
