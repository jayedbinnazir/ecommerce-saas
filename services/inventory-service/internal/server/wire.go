package server

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/inventory-service/internal/auth/token"
	"github.com/jayedbinnazir/inventory-service/internal/authz"
	"github.com/jayedbinnazir/inventory-service/internal/middleware"
	stockroutes "github.com/jayedbinnazir/inventory-service/internal/stock/routes"
)

// registerModules builds the auth guard and mounts every feature module.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	stockroutes.Register(api, s.db, guards, s.cfg.InternalKey)

	return nil
}
