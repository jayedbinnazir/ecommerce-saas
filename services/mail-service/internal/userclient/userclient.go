// Package userclient resolves a user's name + email from user-management, so
// mail-service can address a transactional email off a Kafka event.
package userclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrUpstream = errors.New("user-management is unavailable")

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

func New(baseURL, internalKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     internalKey,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Enabled reports whether a user-management URL is configured.
func (c *Client) Enabled() bool { return c.baseURL != "" }

func (c *Client) Get(ctx context.Context, userID string) (*User, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("X-Internal-Key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	return &body.Data, nil
}
