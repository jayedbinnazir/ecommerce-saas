// Package services holds the auth application logic: local sign-up/sign-in,
// the Google/Facebook OAuth flow, session issuing and refresh-token rotation.
package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/dto"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/oauth"
	authrepo "github.com/jayedbinnazir/golang-saas.git/internal/auth/repository"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/session"
	"github.com/jayedbinnazir/golang-saas.git/internal/auth/token"
	"github.com/jayedbinnazir/golang-saas.git/internal/mailclient"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	userdomain "github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
	userrepo "github.com/jayedbinnazir/golang-saas.git/internal/user/repository"
)

// passwordResetTTL is how long a forgot-password token stays valid.
const passwordResetTTL = 30 * time.Minute

type Service struct {
	db          *sql.DB
	tokens      *token.Manager
	sessions    *session.Store
	providers   *oauth.Registry
	refreshTTL  time.Duration
	mail        *mailclient.Client
	frontendURL string
}

func New(
	db *sql.DB,
	tokens *token.Manager,
	sessions *session.Store,
	providers *oauth.Registry,
	refreshTTL time.Duration,
	mail *mailclient.Client,
	frontendURL string,
) *Service {
	return &Service{
		db:          db,
		tokens:      tokens,
		sessions:    sessions,
		providers:   providers,
		refreshTTL:  refreshTTL,
		mail:        mail,
		frontendURL: strings.TrimRight(frontendURL, "/"),
	}
}

// ---------------------------------------------------------------------
// Local email + password
// ---------------------------------------------------------------------

func (s *Service) Register(ctx context.Context, req dto.RegisterRequest, agent, ip string) (*dto.Tokens, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	var phone *string
	if req.Phone != nil {
		if p := strings.TrimSpace(*req.Phone); p != "" {
			phone = &p
		}
	}

	user := &userdomain.User{Name: name, Email: email, Phone: phone, PasswordHash: &hashStr}
	if err := userrepo.NewUserRepository(s.db).Create(ctx, user); err != nil {
		return nil, err
	}
	return s.issueSession(ctx, s.db, user, agent, ip)
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest, agent, ip string) (*dto.Tokens, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := userrepo.NewUserRepository(s.db).GetByEmail(ctx, email)
	if err != nil {
		return nil, userdomain.ErrInvalidCredentials
	}
	if user.PasswordHash == nil {
		return nil, userdomain.ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)) != nil {
		return nil, userdomain.ErrInvalidCredentials
	}
	return s.issueSession(ctx, s.db, user, agent, ip)
}

// ---------------------------------------------------------------------
// OAuth (Google / Facebook) — backend redirect flow
// ---------------------------------------------------------------------

// StartOAuth returns the provider consent URL to redirect the browser to.
// redirectTo is where we send the user after a successful callback.
func (s *Service) StartOAuth(ctx context.Context, providerName, redirectTo string) (string, error) {
	provider, ok := authdomain.ParseProvider(providerName)
	if !ok {
		return "", authdomain.ErrInvalidProvider
	}
	p, err := s.providers.Get(provider)
	if err != nil {
		return "", err
	}

	state := randomToken()
	if err := s.sessions.SaveState(ctx, state, redirectTo); err != nil {
		return "", err
	}
	return p.AuthCodeURL(state), nil
}

// CompleteOAuth handles the provider callback and returns the issued tokens plus
// the post-login redirect target.
func (s *Service) CompleteOAuth(ctx context.Context, providerName, code, state, agent, ip string) (*dto.Tokens, string, error) {
	provider, ok := authdomain.ParseProvider(providerName)
	if !ok {
		return nil, "", authdomain.ErrInvalidProvider
	}
	p, err := s.providers.Get(provider)
	if err != nil {
		return nil, "", err
	}

	redirectTo, err := s.sessions.ConsumeState(ctx, state)
	if err != nil {
		return nil, "", err
	}

	info, err := p.Exchange(ctx, code)
	if err != nil {
		return nil, "", err
	}
	if info.Email == "" || !info.EmailVerified {
		return nil, "", authdomain.ErrEmailNotVerified
	}

	var tokens *dto.Tokens
	err = platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		user, err := s.findOrCreateOAuthUser(ctx, tx, provider, info)
		if err != nil {
			return err
		}
		tokens, err = s.issueSession(ctx, tx, user, agent, ip)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	return tokens, redirectTo, nil
}

// findOrCreateOAuthUser links an existing account or creates a new one.
func (s *Service) findOrCreateOAuthUser(ctx context.Context, tx *sql.Tx, provider authdomain.Provider, info *oauth.UserInfo) (*userdomain.User, error) {
	identities := authrepo.NewIdentityRepository(tx)
	users := userrepo.NewUserRepository(tx)

	// 1. Known provider identity -> existing user.
	identity, err := identities.GetByProviderSubject(ctx, provider, info.Subject)
	if err == nil {
		return users.GetByID(ctx, identity.UserID)
	}
	if err != authdomain.ErrIdentityNotFound {
		return nil, err
	}

	// 2. No identity yet. Reuse a user with the same email, or create one.
	email := strings.ToLower(strings.TrimSpace(info.Email))
	user, err := users.GetByEmail(ctx, email)
	if err == userdomain.ErrUserNotFound {
		user = &userdomain.User{Name: strings.TrimSpace(info.Name), Email: email} // no password
		if err := users.Create(ctx, user); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// 3. Attach the provider identity to that user.
	providerEmail := info.Email
	if err := identities.Create(ctx, &authdomain.AuthIdentity{
		UserID:          user.ID,
		Provider:        provider,
		ProviderSubject: info.Subject,
		ProviderEmail:   &providerEmail,
	}); err != nil {
		return nil, err
	}
	return user, nil
}

// ---------------------------------------------------------------------
// Password change / forgot / reset
// ---------------------------------------------------------------------

// ChangePassword verifies the caller's current password, rejects reuse of
// that same password, then revokes every session (this one included, so the
// client cookies the handler clears are consistent with the server-side
// state) — same "password changed -> everyone re-authenticates" rule reset
// uses.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	users := userrepo.NewUserRepository(s.db)
	user, err := users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword)) != nil {
		return authdomain.ErrCurrentPasswordInvalid
	}
	if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(newPassword)) == nil {
		return authdomain.ErrPasswordReused
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashStr := string(hash)
	user.PasswordHash = &hashStr
	if err := users.Update(ctx, user); err != nil {
		return err
	}

	return s.revokeAllSessions(ctx, userID)
}

// RequestPasswordReset issues a single-use reset token and emails it, if the
// address belongs to a real, password-having account. It never reports which
// case applies (unknown email, OAuth-only account, or mail delivery failure)
// — the caller always sees the same outcome.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := userrepo.NewUserRepository(s.db).GetByEmail(ctx, email)
	if err != nil {
		return nil // unknown email -- do not reveal
	}
	if user.PasswordHash == nil {
		return nil // OAuth-only account has nothing to reset
	}

	raw := randomToken()
	reset := &authdomain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().UTC().Add(passwordResetTTL),
	}
	if err := authrepo.NewPasswordResetRepository(s.db).Create(ctx, reset); err != nil {
		return err
	}

	if s.mail != nil && s.mail.Enabled() {
		resetURL := s.frontendURL + "/reset-password?token=" + raw
		if err := s.mail.Send(ctx, user.Email, "password_reset", map[string]any{
			"name":      user.Name,
			"reset_url": resetURL,
			"token":     raw,
		}); err != nil {
			log.Printf("[auth] password reset email to %s: %v", user.Email, err)
		}
	}
	return nil
}

// ResetPassword consumes a single-use token and sets a new password. The
// token row is locked (FOR UPDATE) for the whole check-and-consume so a
// concurrent double-submit of the same token can't both succeed.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hash := hashToken(rawToken)

	var userID uuid.UUID
	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		resetRepo := authrepo.NewPasswordResetRepository(tx)

		reset, err := resetRepo.GetByHashForUpdate(ctx, hash)
		if err != nil {
			return authdomain.ErrResetTokenInvalid
		}
		if reset.IsUsed() || reset.IsExpired(time.Now().UTC()) {
			return authdomain.ErrResetTokenInvalid
		}
		if err := resetRepo.MarkUsed(ctx, reset.ID); err != nil {
			return err
		}

		users := userrepo.NewUserRepository(tx)
		user, err := users.GetByID(ctx, reset.UserID)
		if err != nil {
			return err
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		hashStr := string(hashed)
		user.PasswordHash = &hashStr
		if err := users.Update(ctx, user); err != nil {
			return err
		}

		userID = user.ID
		return nil
	})
	if err != nil {
		return err
	}

	return s.revokeAllSessions(ctx, userID)
}

// revokeAllSessions revokes every refresh token for the user (every device)
// and drops the matching Redis sessions, so every previously-issued access
// token stops working on its very next request.
func (s *Service) revokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	sessionIDs, err := authrepo.NewRefreshTokenRepository(s.db).RevokeAllForUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, sid := range sessionIDs {
		_ = s.sessions.Delete(ctx, sid)
	}
	return nil
}

// ---------------------------------------------------------------------
// Refresh + logout
// ---------------------------------------------------------------------

// Refresh rotates a refresh token: the old one is revoked and a new pair issued.
// Presenting an already-revoked token revokes the whole family (reuse detection).
func (s *Service) Refresh(ctx context.Context, rawRefresh, agent, ip string) (*dto.Tokens, error) {
	hash := hashToken(rawRefresh)
	refreshRepo := authrepo.NewRefreshTokenRepository(s.db)

	current, err := refreshRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, authdomain.ErrRefreshInvalid
	}

	if current.IsRevoked() {
		_ = refreshRepo.RevokeFamily(ctx, current.FamilyID)
		_ = s.sessions.Delete(ctx, current.SessionID)
		return nil, authdomain.ErrRefreshInvalid
	}
	if current.IsExpired(time.Now().UTC()) {
		return nil, authdomain.ErrRefreshInvalid
	}

	users := userrepo.NewUserRepository(s.db)
	user, err := users.GetByID(ctx, current.UserID)
	if err != nil {
		return nil, err
	}

	var tokens *dto.Tokens
	err = platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		txRefresh := authrepo.NewRefreshTokenRepository(tx)
		if err := txRefresh.RevokeByID(ctx, current.ID); err != nil {
			return err
		}
		tokens, err = s.issueSessionInFamily(ctx, tx, user, current.SessionID, current.FamilyID, agent, ip)
		return err
	})
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// Logout revokes every refresh token for the session and drops the Redis record.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if err := authrepo.NewRefreshTokenRepository(s.db).RevokeSession(ctx, sessionID); err != nil {
		return err
	}
	return s.sessions.Delete(ctx, sessionID)
}

// ---------------------------------------------------------------------
// session issuing
// ---------------------------------------------------------------------

func (s *Service) issueSession(ctx context.Context, exec platform.DBTX, user *userdomain.User, agent, ip string) (*dto.Tokens, error) {
	return s.issueSessionInFamily(ctx, exec, user, uuid.New(), uuid.New(), agent, ip)
}

func (s *Service) issueSessionInFamily(
	ctx context.Context,
	exec platform.DBTX,
	user *userdomain.User,
	sessionID, familyID uuid.UUID,
	agent, ip string,
) (*dto.Tokens, error) {
	now := time.Now().UTC()

	rawRefresh := randomToken()
	refresh := &authdomain.RefreshToken{
		UserID:    user.ID,
		SessionID: sessionID,
		FamilyID:  familyID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: now.Add(s.refreshTTL),
		UserAgent: nullable(agent),
		IPAddress: nullable(ip),
	}
	if err := authrepo.NewRefreshTokenRepository(exec).Create(ctx, refresh); err != nil {
		return nil, err
	}

	if err := s.sessions.Save(ctx, sessionID, session.Session{
		UserID:     user.ID,
		SuperAdmin: user.IsSuperAdmin,
	}); err != nil {
		return nil, err
	}

	access, err := s.tokens.Issue(user.ID, sessionID, user.IsSuperAdmin)
	if err != nil {
		return nil, err
	}

	return dto.NewTokens(access, rawRefresh, s.tokens.TTL(), s.refreshTTL), nil
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func nullable(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
