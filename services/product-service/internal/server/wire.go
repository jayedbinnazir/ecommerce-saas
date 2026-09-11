package server

import (
	"github.com/gin-gonic/gin"

	attributeroutes "github.com/jayedbinnazir/product-service/internal/attributes/routes"
	"github.com/jayedbinnazir/product-service/internal/auth/token"
	"github.com/jayedbinnazir/product-service/internal/authz"
	categoryroutes "github.com/jayedbinnazir/product-service/internal/category/routes"
	"github.com/jayedbinnazir/product-service/internal/middleware"
	productroutes "github.com/jayedbinnazir/product-service/internal/product/routes"
	reviewroutes "github.com/jayedbinnazir/product-service/internal/review/routes"
)

// registerModules builds the auth guard and mounts every feature module.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	categoryroutes.Register(api, s.db, guards)
	attributeroutes.Register(api, s.db, guards)
	productroutes.Register(api, s.db, s.storage, guards)
	reviewroutes.Register(api, s.db, guards)

	return nil
}
