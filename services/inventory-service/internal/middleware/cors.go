package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows the given origins to call the API with credentials. Empty entries
// are skipped.
func CORS(origins ...string) gin.HandlerFunc {
	allowed := make([]string, 0, len(origins))
	for _, o := range origins {
		if o != "" {
			allowed = append(allowed, o)
		}
	}
	if len(allowed) == 0 {
		allowed = []string{"http://localhost:3000"}
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowed,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", RequestIDHeader},
		ExposeHeaders:    []string{"Content-Length", RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
