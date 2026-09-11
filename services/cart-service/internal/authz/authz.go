// Package authz asks the user-management service which tenants the caller
// belongs to and with what role. Results are cached briefly per token.
package authz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Role names, matching user-management.
const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleAdmin      = "ADMIN"
	RoleManager    = "MANAGER"
	RoleCustomer   = "CUSTOMER"
)

var (
	// ErrUnauthorized means user-management rejected the token.
	ErrUnauthorized = errors.New("access token rejected by user-management")
	// ErrUpstream means user-management was unreachable or errored.
	ErrUpstream = errors.New("user-management is unavailable")
)

// Membership is the caller's role in one tenant.
type Membership struct {
	TenantID uuid.UUID
	Role     string
}

type Client struct {
	baseURL string
	http    *http.Client

	mu    sync.Mutex
	cache map[string]cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	memberships []Membership
	expiresAt   time.Time
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
		cache:   make(map[string]cacheEntry),
		ttl:     30 * time.Second,
	}
}

// RoleInTenant returns the caller's role in tenantID, or "" if they are not a
// member. rawToken is the caller's access token.
func (c *Client) RoleInTenant(ctx context.Context, rawToken string, tenantID uuid.UUID) (string, error) {
	memberships, err := c.memberships(ctx, rawToken)
	if err != nil {
		return "", err
	}
	for _, m := range memberships {
		if m.TenantID == tenantID {
			return m.Role, nil
		}
	}
	return "", nil
}

func (c *Client) memberships(ctx context.Context, rawToken string) ([]Membership, error) {
	key := tokenKey(rawToken)

	c.mu.Lock()
	if entry, ok := c.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.memberships, nil
	}
	c.mu.Unlock()

	memberships, err := c.fetchMemberships(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = cacheEntry{memberships: memberships, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()

	return memberships, nil
}

func (c *Client) fetchMemberships(ctx context.Context, rawToken string) ([]Membership, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/me/memberships", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrUnauthorized
	default:
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data []struct {
			TenantID uuid.UUID `json:"tenant_id"`
			RoleName string    `json:"role_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	out := make([]Membership, 0, len(body.Data))
	for _, m := range body.Data {
		out = append(out, Membership{TenantID: m.TenantID, Role: m.RoleName})
	}
	return out, nil
}

func tokenKey(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
