// Package inventoryclient calls inventory-service's internal bulk stock API. It
// authenticates with a shared X-Internal-Key, not a user token.
package inventoryclient

import (
	"bytes"
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
	// ErrStockConflict means inventory-service refused the change (e.g. not
	// enough available stock). The wrapped message explains which SKU.
	ErrStockConflict = errors.New("inventory rejected the stock change")
	// ErrUpstream means inventory-service was unreachable or errored.
	ErrUpstream = errors.New("inventory-service is unavailable")
)

// Line is one SKU + quantity.
type Line struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

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

// Reserve holds stock for an order. reference is the order id.
func (c *Client) Reserve(ctx context.Context, tenantID uuid.UUID, reference string, lines []Line) error {
	return c.post(ctx, "reserve", tenantID, reference, lines)
}

// Release returns previously reserved stock (order cancelled).
func (c *Client) Release(ctx context.Context, tenantID uuid.UUID, reference string, lines []Line) error {
	return c.post(ctx, "release", tenantID, reference, lines)
}

// Ship removes reserved stock from on-hand (order fulfilled).
func (c *Client) Ship(ctx context.Context, tenantID uuid.UUID, reference string, lines []Line) error {
	return c.post(ctx, "ship", tenantID, reference, lines)
}

// Restock puts returned units back on the shelf (on_hand += qty).
func (c *Client) Restock(ctx context.Context, tenantID uuid.UUID, reference string, lines []Line) error {
	return c.post(ctx, "restock", tenantID, reference, lines)
}

func (c *Client) post(ctx context.Context, op string, tenantID uuid.UUID, reference string, lines []Line) error {
	url := fmt.Sprintf("%s/api/v1/internal/tenants/%s/stock/%s", c.baseURL, tenantID, op)

	payload, err := json.Marshal(struct {
		Reference string `json:"reference"`
		Items     []Line `json:"items"`
	}{Reference: reference, Items: lines})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrStockConflict, errorMessage(resp))
	default:
		return fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
}

// errorMessage pulls the human message out of the standard error envelope.
func errorMessage(resp *http.Response) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Error.Message == "" {
		return "stock unavailable"
	}
	return body.Error.Message
}
