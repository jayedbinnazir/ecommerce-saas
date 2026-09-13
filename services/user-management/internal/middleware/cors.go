package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows the frontend origin (and the API's own origin) to call the API
// with cookies. Pass the configured frontend URL; empty is ignored.
func CORS(frontendURL string) gin.HandlerFunc {
	origins := []string{"http://localhost:8080"}
	if frontendURL != "" {
		origins = append(origins, frontendURL)
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", RequestIDHeader},
		ExposeHeaders:    []string{"Content-Length", RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
