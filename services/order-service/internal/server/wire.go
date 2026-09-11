package server

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/order-service/internal/auth/token"
	"github.com/jayedbinnazir/order-service/internal/authz"
	"github.com/jayedbinnazir/order-service/internal/cartclient"
	"github.com/jayedbinnazir/order-service/internal/inventoryclient"
	"github.com/jayedbinnazir/order-service/internal/middleware"
	orderroutes "github.com/jayedbinnazir/order-service/internal/order/routes"
	"github.com/jayedbinnazir/order-service/internal/paymentclient"
	returnroutes "github.com/jayedbinnazir/order-service/internal/returns/routes"
	storeconfigrepo "github.com/jayedbinnazir/order-service/internal/storeconfig/repository"
	storeconfigroutes "github.com/jayedbinnazir/order-service/internal/storeconfig/routes"
	storeconfig "github.com/jayedbinnazir/order-service/internal/storeconfig/services"
)

// registerModules builds the auth guard, the downstream clients and mounts every
// feature module.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	cartClient := cartclient.NewClient(s.cfg.Services.CartServiceURL)
	inventoryClient := inventoryclient.NewClient(s.cfg.Services.InventoryServiceURL, s.cfg.Services.InventoryInternalKey)
	paymentClient := paymentclient.NewClient(s.cfg.Services.PaymentServiceURL, s.cfg.Services.PaymentInternalKey)

	storeCfg := storeconfig.New(storeconfigrepo.New(s.db))
	storeconfigroutes.Register(api, storeCfg, guards)

	orderroutes.Register(api, s.db, cartClient, inventoryClient, paymentClient, storeCfg, s.events, authzClient, guards)
	returnroutes.Register(api, s.db, paymentClient, inventoryClient, authzClient, guards)

	return nil
}
