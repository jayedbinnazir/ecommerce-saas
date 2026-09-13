// Package middleware holds the service-wide HTTP middleware: CORS, request id,
// panic recovery, rate limiting and authentication.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/golang-saas.git/internal/auth/session"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/token"
	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

// AccessCookieName is the cookie the access token is stored in. The auth handler
// writes it; this middleware reads it on every request.
const AccessCookieName = "access_token"

// AuthGuard validates the access token + Redis session on protected routes.
type AuthGuard struct {
	tokens   *token.Manager
	sessions *session.Store
}

func NewAuthGuard(tokens *token.Manager, sessions *session.Store) *AuthGuard {
	return &AuthGuard{tokens: tokens, sessions: sessions}
}

// Guards returns the bundle handed to every feature module's route registrar.
func (g *AuthGuard) Guards() httpx.Guards {
	return httpx.Guards{
		Authenticated: g.Authenticated,
		SuperAdmin:    g.SuperAdmin,
	}
}

// Authenticated rejects the request (401) unless a valid session is present, and
// puts the platform.Principal on the request context.
func (g *AuthGuard) Authenticated(c *gin.Context) {
	raw := g.readToken(c)
	if raw == "" {
		g.reject(c, httpx.Unauthorized("authentication required"))
		return
	}

	claims, err := g.tokens.Verify(raw)
	if err != nil {
		g.reject(c, httpx.Unauthorized("invalid or expired access token"))
		return
	}

	userID, err := claims.UserID()
	if err != nil {
		g.reject(c, httpx.Unauthorized("malformed token"))
		return
	}
	sessionID, err := claims.Session()
	if err != nil {
		g.reject(c, httpx.Unauthorized("malformed token"))
		return
	}

	sess, err := g.sessions.Get(c.Request.Context(), sessionID)
	if err != nil {
		g.reject(c, httpx.Unauthorized("session is no longer valid"))
		return
	}

	principal := platform.Principal{
		UserID:       userID,
		SessionID:    sessionID,
		IsSuperAdmin: sess.SuperAdmin,
	}
	c.Request = c.Request.WithContext(platform.WithPrincipal(c.Request.Context(), principal))
	c.Next()
}

// SuperAdmin must run after Authenticated.
func (g *AuthGuard) SuperAdmin(c *gin.Context) {
	principal, ok := platform.PrincipalFromContext(c.Request.Context())
	if !ok {
		g.reject(c, httpx.Unauthorized("authentication required"))
		return
	}
	if !principal.IsSuperAdmin {
		g.reject(c, httpx.Forbidden("super-admin access required"))
		return
	}
	c.Next()
}

// readToken looks first at the access-token cookie, then a Bearer header.
func (g *AuthGuard) readToken(c *gin.Context) string {
	if cookie, err := c.Cookie(AccessCookieName); err == nil && cookie != "" {
		return cookie
	}
	if after, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); found {
		return strings.TrimSpace(after)
	}
	return ""
}

func (g *AuthGuard) reject(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
