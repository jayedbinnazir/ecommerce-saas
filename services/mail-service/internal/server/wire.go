package server

import (
	"github.com/gin-gonic/gin"

	mailroutes "github.com/jayedbinnazir/mail-service/internal/mail/routes"
)

func (s *Server) registerModules(api *gin.RouterGroup) error {
	mailroutes.Register(api, s.db, s.sender, s.cfg.InternalKey)
	return nil
}
