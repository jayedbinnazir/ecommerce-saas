// Package availability is a thin client for inventory-service. cart-service uses
// it to check there is enough stock before adding or increasing a line.
package availability

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

// ErrUpstream means inventory-service was unreachable or errored.
var ErrUpstream = errors.New("inventory-service is unavailable")

// Stock is the availability of one sku. Tracked is false when inventory-service
// has no record of the sku — such items are treated as unlimited.
type Stock struct {
	Available int
	Tracked   bool
}

// Enough reports whether want units can be sold.
func (s Stock) Enough(want int) bool { return !s.Tracked || s.Available >= want }

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Get returns the current availability of sku in tenantID. rawToken is forwarded
// though the endpoint is public.
func (c *Client) Get(ctx context.Context, rawToken string, tenantID uuid.UUID, sku string) (Stock, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/stock/%s", c.baseURL, tenantID, sku)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Stock{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if rawToken != "" {
		req.Header.Set("Authorization", "Bearer "+rawToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Stock{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return Stock{Tracked: false}, nil
	default:
		return Stock{}, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data struct {
			Available int `json:"available"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Stock{}, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	return Stock{Available: body.Data.Available, Tracked: true}, nil
}
