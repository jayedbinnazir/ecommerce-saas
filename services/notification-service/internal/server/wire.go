package server

import (
	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/notification-service/internal/auth/token"
	"github.com/jayedbinnazir/notification-service/internal/authz"
	"github.com/jayedbinnazir/notification-service/internal/mailclient"
	"github.com/jayedbinnazir/notification-service/internal/middleware"
	notifroutes "github.com/jayedbinnazir/notification-service/internal/notification/routes"
)

func (s *Server) registerModules(api *gin.RouterGroup) error {
	verifier := token.NewVerifier(s.cfg.JWT.Secret)
	authzClient := authz.NewClient(s.cfg.Services.UserServiceURL)
	guards := middleware.NewGuard(verifier, authzClient).Guards()

	mail := mailclient.New(s.cfg.Services.MailServiceURL, s.cfg.Services.MailInternalKey)

	notifroutes.Register(api, s.db, mail, guards, s.cfg.InternalKey)

	return nil
}
