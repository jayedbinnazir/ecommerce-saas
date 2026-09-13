// Package oauth wraps the Google and Facebook OAuth2 "authorization code" flow.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	config "github.com/jayedbinnazir/golang-saas.git/internal/config"
)

// UserInfo is the normalized profile we get back from a provider.
type UserInfo struct {
	Subject       string
	Email         string
	Name          string
	EmailVerified bool
}

// Provider knows how to build a consent URL and turn a code into a UserInfo.
type Provider struct {
	Name        authdomain.Provider
	oauthConfig *oauth2.Config
	userInfoURL string
}

// AuthCodeURL is the URL to send the browser to for consent.
func (p *Provider) AuthCodeURL(state string) string {
	return p.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange swaps the callback code for a token and fetches the user's profile.
func (p *Provider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", authdomain.ErrOAuthExchange, err)
	}

	client := p.oauthConfig.Client(ctx, token)
	client.Timeout = 10 * time.Second

	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", authdomain.ErrOAuthExchange, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: userinfo status %d", authdomain.ErrOAuthExchange, resp.StatusCode)
	}

	switch p.Name {
	case authdomain.ProviderGoogle:
		return decodeGoogle(resp.Body)
	case authdomain.ProviderFacebook:
		return decodeFacebook(resp.Body)
	default:
		return nil, authdomain.ErrInvalidProvider
	}
}

// Registry holds every configured provider.
type Registry struct {
	providers map[authdomain.Provider]*Provider
}

// NewRegistry builds providers from config. A provider with an empty client id
// is simply skipped (and requests for it later return ErrProviderNotConfig).
func NewRegistry(cfg config.AuthConfig) *Registry {
	r := &Registry{providers: map[authdomain.Provider]*Provider{}}

	if cfg.Google.ClientID != "" {
		r.providers[authdomain.ProviderGoogle] = &Provider{
			Name:        authdomain.ProviderGoogle,
			userInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
			oauthConfig: &oauth2.Config{
				ClientID:     cfg.Google.ClientID,
				ClientSecret: cfg.Google.ClientSecret,
				RedirectURL:  cfg.Google.RedirectURL,
				Endpoint:     endpoints.Google,
				Scopes:       []string{"openid", "email", "profile"},
			},
		}
	}

	if cfg.Facebook.ClientID != "" {
		r.providers[authdomain.ProviderFacebook] = &Provider{
			Name:        authdomain.ProviderFacebook,
			userInfoURL: "https://graph.facebook.com/me?fields=id,name,email",
			oauthConfig: &oauth2.Config{
				ClientID:     cfg.Facebook.ClientID,
				ClientSecret: cfg.Facebook.ClientSecret,
				RedirectURL:  cfg.Facebook.RedirectURL,
				Endpoint:     endpoints.Facebook,
				Scopes:       []string{"email", "public_profile"},
			},
		}
	}

	return r
}

// Get returns the provider or ErrProviderNotConfig.
func (r *Registry) Get(name authdomain.Provider) (*Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, authdomain.ErrProviderNotConfig
	}
	return p, nil
}

// ---- provider-specific response shapes ----

func decodeGoogle(body io.Reader) (*UserInfo, error) {
	var raw struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: %v", authdomain.ErrOAuthExchange, err)
	}
	return &UserInfo{
		Subject:       raw.Sub,
		Email:         raw.Email,
		Name:          raw.Name,
		EmailVerified: raw.EmailVerified,
	}, nil
}

func decodeFacebook(body io.Reader) (*UserInfo, error) {
	var raw struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("%w: %v", authdomain.ErrOAuthExchange, err)
	}
	// Facebook only returns an email if the account has a confirmed one.
	return &UserInfo{
		Subject:       raw.ID,
		Email:         raw.Email,
		Name:          raw.Name,
		EmailVerified: raw.Email != "",
	}, nil
}
