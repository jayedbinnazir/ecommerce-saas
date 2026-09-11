// Package middleware holds the service-wide HTTP middleware: CORS, request id,
// panic recovery, rate limiting and authentication.
package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jayedbinnazir/product-service/internal/auth/token"
	"github.com/jayedbinnazir/product-service/internal/authz"
	"github.com/jayedbinnazir/product-service/internal/httpx"
	"github.com/jayedbinnazir/product-service/internal/platform"
)

// AccessCookieName is the cookie the user-management service stores the access
// token in; product-service also accepts it as a Bearer header.
const AccessCookieName = "access_token"

// Guard authenticates requests and enforces tenant-scoped roles. It verifies the
// access token locally (shared secret) and asks user-management for memberships.
type Guard struct {
	verifier *token.Verifier
	authz    *authz.Client
}

func NewGuard(verifier *token.Verifier, authzClient *authz.Client) *Guard {
	return &Guard{verifier: verifier, authz: authzClient}
}

// Guards is the bundle handed to every feature module's route registrar.
func (g *Guard) Guards() httpx.Guards {
	return httpx.Guards{
		Authenticated: g.Authenticated,
		SuperAdmin:    g.SuperAdmin,
		TenantRole:    g.RequireTenantRole,
	}
}

// Authenticated verifies the access token and stores the platform.Principal
// (including the raw token, for downstream authz calls) on the request context.
func (g *Guard) Authenticated(c *gin.Context) {
	raw := readToken(c)
	if raw == "" {
		reject(c, httpx.Unauthorized("authentication required"))
		return
	}

	claims, err := g.verifier.Verify(raw)
	if err != nil {
		reject(c, httpx.Unauthorized("invalid or expired access token"))
		return
	}

	userID, err := claims.UserID()
	if err != nil {
		reject(c, httpx.Unauthorized("malformed token"))
		return
	}
	sessionID, _ := claims.Session()

	principal := platform.Principal{
		UserID:       userID,
		SessionID:    sessionID,
		IsSuperAdmin: claims.SuperAdmin,
		RawToken:     raw,
	}
	c.Request = c.Request.WithContext(platform.WithPrincipal(c.Request.Context(), principal))
	c.Next()
}

// SuperAdmin must run after Authenticated.
func (g *Guard) SuperAdmin(c *gin.Context) {
	principal, ok := platform.PrincipalFromContext(c.Request.Context())
	if !ok {
		reject(c, httpx.Unauthorized("authentication required"))
		return
	}
	if !principal.IsSuperAdmin {
		reject(c, httpx.Forbidden("super-admin access required"))
		return
	}
	c.Next()
}

// RequireTenantRole allows super admins, or callers whose membership in the
// tenant named by tenantParam has one of the given roles. Must run after
// Authenticated.
func (g *Guard) RequireTenantRole(tenantParam string, roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		principal, ok := httpx.Principal(c)
		if !ok {
			c.Abort()
			return
		}
		if principal.IsSuperAdmin {
			c.Next()
			return
		}

		tenantID, ok := httpx.UUIDParam(c, tenantParam)
		if !ok {
			c.Abort()
			return
		}

		role, err := g.authz.RoleInTenant(c.Request.Context(), principal.RawToken, tenantID)
		switch {
		case errors.Is(err, authz.ErrUnauthorized):
			reject(c, httpx.Unauthorized("session is no longer valid"))
			return
		case errors.Is(err, authz.ErrUpstream):
			reject(c, httpx.Upstream("could not verify tenant membership"))
			return
		case err != nil:
			reject(c, httpx.Internal("authorization check failed").Wrap(err))
			return
		}

		if role == "" {
			reject(c, httpx.Forbidden("you are not a member of this tenant"))
			return
		}
		if _, ok := allowed[role]; !ok {
			reject(c, httpx.Forbidden("insufficient role for this action"))
			return
		}
		c.Next()
	}
}

func readToken(c *gin.Context) string {
	if cookie, err := c.Cookie(AccessCookieName); err == nil && cookie != "" {
		return cookie
	}
	if after, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); found {
		return strings.TrimSpace(after)
	}
	return ""
}

func reject(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}
