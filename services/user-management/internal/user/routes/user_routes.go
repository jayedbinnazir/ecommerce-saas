// Package routes wires the user module and registers it on a Gin router group.
package routes

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/middleware"
	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/user/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/repository"
	"github.com/jayedbinnazir/golang-saas.git/internal/user/services"
)

// Register mounts:
//   - /users            platform user administration (super-admin only)
//   - /users/:userId/addresses   a user's own shipping/billing addresses
func Register(rg *gin.RouterGroup, db *sql.DB, guards httpx.Guards, internalKey string) {
	userRepo := repository.NewUserRepository(db)

	userSvc := services.NewUserService(userRepo)
	addressSvc := services.NewAddressService(db, userRepo)

	userH := httphandler.NewUserHandler(userSvc)
	addressH := httphandler.NewAddressHandler(addressSvc)

	admin := rg.Group("/users", guards.Authenticated, guards.SuperAdmin)
	{
		admin.GET("", userH.List)
		admin.GET("/:userId", userH.Get)
		admin.PATCH("/:userId", userH.Update)
		admin.DELETE("/:userId", userH.Delete)
	}

	// service-to-service: resolve a user's name + email
	rg.GET("/internal/users/:userId", middleware.RequireInternalKey(internalKey), userH.GetInternal)

	addresses := rg.Group("/users/:userId/addresses", guards.Authenticated)
	{
		addresses.POST("", addressH.Create)
		addresses.GET("", addressH.List)
		addresses.GET("/:addressId", addressH.Get)
		addresses.PATCH("/:addressId", addressH.Update)
		addresses.DELETE("/:addressId", addressH.Delete)
	}
}
