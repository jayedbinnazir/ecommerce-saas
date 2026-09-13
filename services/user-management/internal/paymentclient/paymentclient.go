// Package paymentclient asks payment-service whether a user has an active
// subscription. It implements the tenant module's SubscriptionGuard so store
// creation stays gated even though billing now lives in another service.
package paymentclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrSubscriptionRequired is returned when the user has no active plan.
	ErrSubscriptionRequired = errors.New("an active subscription is required to create a store")
	// ErrUpstream means payment-service was unreachable or errored.
	ErrUpstream = errors.New("payment-service is unavailable")
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewClient(baseURL, internalKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     internalKey,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// RequireActiveSubscription implements tenant/domain.SubscriptionGuard.
func (c *Client) RequireActiveSubscription(ctx context.Context, userID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/internal/users/%s/subscription", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("X-Internal-Key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data struct {
			Active bool `json:"active"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if !body.Data.Active {
		return ErrSubscriptionRequired
	}
	return nil
}
