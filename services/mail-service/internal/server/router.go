package server

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/mail-service/internal/httpx"
	"github.com/jayedbinnazir/mail-service/internal/middleware"
)

func (s *Server) RouterSetup() (*gin.Engine, error) {
	if os.Getenv("APP_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	httpx.ConfigureValidator()

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.CORS("http://localhost:3000", "http://localhost:5173"),
		gin.Logger(),
		middleware.Recovery(),
		middleware.RateLimit(30, 60),
		httpx.ErrorHandler(),
	)

	api := router.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "API is healthy"})
	})

	if err := s.registerModules(api); err != nil {
		return nil, err
	}
	return router, nil
}
