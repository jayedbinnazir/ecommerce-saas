// Package httphandler contains the Gin HTTP handlers for the auth module. It is
// the only place that reads/writes auth cookies.
package httphandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/services"
	"github.com/jayedbinnazir/golang-saas.git/internal/httpx"
	"github.com/jayedbinnazir/golang-saas.git/internal/middleware"
	userdomain "github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
	userdto "github.com/jayedbinnazir/golang-saas.git/internal/user/dto"
)

const refreshCookieName = "refresh_token"
const refreshCookiePath = "/api/v1/auth"

// CookieConfig controls how auth cookies are written.
type CookieConfig struct {
	Domain      string
	Secure      bool
	FrontendURL string // fallback redirect target for OAuth
}

type Handler struct {
	svc    *services.Service
	users  userdomain.UserRepository
	cookie CookieConfig
}

func New(svc *services.Service, users userdomain.UserRepository, cookie CookieConfig) *Handler {
	return &Handler{svc: svc, users: users, cookie: cookie}
}

// ---- local ----

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, err := h.svc.Register(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.writeCookies(c, tokens)
	httpx.Created(c, tokens)
}

func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, err := h.svc.Login(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.writeCookies(c, tokens)
	httpx.OK(c, tokens)
}

func (h *Handler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookieName)
	if err != nil || raw == "" {
		_ = c.Error(httpx.Unauthorized("missing refresh token"))
		return
	}
	tokens, err := h.svc.Refresh(c.Request.Context(), raw, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.clearCookies(c)
		_ = c.Error(err)
		return
	}
	h.writeCookies(c, tokens)
	httpx.OK(c, tokens)
}

// Logout must run behind the Authenticated guard so the principal is available.
func (h *Handler) Logout(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	if err := h.svc.Logout(c.Request.Context(), principal.SessionID); err != nil {
		_ = c.Error(err)
		return
	}
	h.clearCookies(c)
	httpx.NoContent(c)
}

// ChangePassword requires the caller's current password, rejects reuse of it,
// and revokes every session (including this one) — the client must sign in
// again afterward, so cookies are cleared too.
func (h *Handler) ChangePassword(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	var req dto.ChangePasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), principal.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		_ = c.Error(err)
		return
	}
	h.clearCookies(c)
	httpx.NoContent(c)
}

// ForgotPassword always responds the same way, whether or not the email
// belongs to an account — it must never let a caller distinguish the two.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	_ = h.svc.RequestPasswordReset(c.Request.Context(), req.Email)
	httpx.OK(c, gin.H{"message": "if that email is registered, a reset link has been sent"})
}

// ResetPassword consumes a single-use reset token and sets a new password.
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		_ = c.Error(err)
		return
	}
	httpx.NoContent(c)
}

// Me returns the current user. Runs behind the Authenticated guard.
func (h *Handler) Me(c *gin.Context) {
	principal, ok := httpx.Principal(c)
	if !ok {
		return
	}
	user, err := h.users.GetByID(c.Request.Context(), principal.UserID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	httpx.OK(c, userdto.FromUser(user))
}

// ---- OAuth ----

func (h *Handler) OAuthStart(c *gin.Context) {
	provider := c.Param("provider")
	if _, ok := authdomain.ParseProvider(provider); !ok {
		_ = c.Error(httpx.BadRequest("unknown provider"))
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo == "" {
		redirectTo = h.cookie.FrontendURL
	}

	url, err := h.svc.StartOAuth(c.Request.Context(), provider, redirectTo)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		_ = c.Error(httpx.BadRequest("missing code or state"))
		return
	}

	tokens, redirectTo, err := h.svc.CompleteOAuth(
		c.Request.Context(), provider, code, state, c.Request.UserAgent(), c.ClientIP(),
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	h.writeCookies(c, tokens)
	if redirectTo == "" {
		redirectTo = h.cookie.FrontendURL
	}
	c.Redirect(http.StatusFound, redirectTo)
}

// ---- cookie helpers ----

func (h *Handler) writeCookies(c *gin.Context, tokens *dto.Tokens) {
	c.SetSameSite(http.SameSiteLaxMode)
	// access token: sent on every API call
	c.SetCookie(middleware.AccessCookieName, tokens.AccessToken, tokens.AccessMaxAge(), "/", h.cookie.Domain, h.cookie.Secure, true)
	// refresh token: only sent to the auth endpoints
	c.SetCookie(refreshCookieName, tokens.RefreshToken, tokens.RefreshMaxAge(), refreshCookiePath, h.cookie.Domain, h.cookie.Secure, true)
}

func (h *Handler) clearCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessCookieName, "", -1, "/", h.cookie.Domain, h.cookie.Secure, true)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, h.cookie.Domain, h.cookie.Secure, true)
}
