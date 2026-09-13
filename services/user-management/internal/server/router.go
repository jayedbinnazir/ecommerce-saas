package server

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/middleware"
)

// RouterSetup creates and configures the Gin router and mounts every module.
func (s *Server) RouterSetup() (*gin.Engine, error) {
	if os.Getenv("APP_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Report JSON field names ("email") in validation errors, not Go names.
	httpx.ConfigureValidator()

	router := gin.New()

	// Service-wide middleware (see internal/middleware).
	router.Use(
		middleware.RequestID(),
		middleware.CORS(s.cfg.Auth.FrontendURL),
		gin.Logger(),
		middleware.Recovery(),
		middleware.RateLimit(20, 40), // ~20 req/s per IP, burst 40
		httpx.ErrorHandler(),         // renders anything a handler reports with c.Error
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
