// Package routes wires the auth module: token manager, Redis session store,
// OAuth providers, service, guard and HTTP routes.
package routes

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	httphandler "github.com/jayedbinnazir/golang-saas.git/internal/auth/handler/http"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/oauth"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/services"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/session"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/token"
	config "github.com/jayedbinnazir/golang-saas.git/internal/config"
	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/mailclient"
	"github.com/jayedbinnazir/golang-saas.git/internal/middleware"
	userrepo "github.com/jayedbinnazir/golang-saas.git/internal/user/repository"
)

// Module bundles what the composition root needs from the auth package.
type Module struct {
	Guard   *middleware.AuthGuard
	Handler *httphandler.Handler
}

// Build constructs the whole auth module.
func Build(db *sql.DB, rdb *redis.Client, cfg *config.Config) (*Module, error) {
	accessTTL, err := config.ParseDuration(cfg.JWT.AccessTTL)
	if err != nil {
		return nil, fmt.Errorf("parse jwt.access_ttl %q: %w", cfg.JWT.AccessTTL, err)
	}
	refreshTTL, err := config.ParseDuration(cfg.JWT.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("parse jwt.refresh_ttl %q: %w", cfg.JWT.RefreshTTL, err)
	}

	tokens := token.NewManager(cfg.JWT.Secret, accessTTL)
	sessions := session.NewStore(rdb, refreshTTL)
	providers := oauth.NewRegistry(cfg.Auth)
	mail := mailclient.New(cfg.Services.MailServiceURL, cfg.Services.MailInternalKey)

	svc := services.New(db, tokens, sessions, providers, refreshTTL, mail, cfg.Auth.FrontendURL)

	handler := httphandler.New(svc, userrepo.NewUserRepository(db), httphandler.CookieConfig{
		Domain:      cfg.Auth.CookieDomain,
		Secure:      cfg.Auth.CookieSecure,
		FrontendURL: cfg.Auth.FrontendURL,
	})

	return &Module{
		Guard:   middleware.NewAuthGuard(tokens, sessions),
		Handler: handler,
	}, nil
}

// Register mounts the auth API under rg (typically /api/v1).
func Register(rg *gin.RouterGroup, m *Module, guards httpx.Guards) {
	h := m.Handler

	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", guards.Authenticated, h.Logout)
		auth.GET("/me", guards.Authenticated, h.Me)

		auth.POST("/change-password", guards.Authenticated, h.ChangePassword)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)

		auth.GET("/:provider/login", h.OAuthStart)
		auth.GET("/:provider/callback", h.OAuthCallback)
	}
}
