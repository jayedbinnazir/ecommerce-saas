// Package cartclient is a thin client for cart-service. order-service reads the
// shopper's cart at checkout and clears it once the order is placed.
package cartclient

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
	// ErrEmptyCart means there is nothing to check out.
	ErrEmptyCart = errors.New("cart is empty")
	// ErrUpstream means cart-service was unreachable or errored.
	ErrUpstream = errors.New("cart-service is unavailable")
)

// Line is one cart line, already price-snapshotted by cart-service.
type Line struct {
	ProductID      uuid.UUID
	SKU            string
	Quantity       int
	UnitPriceCents int64
	Currency       string
	ProductName    string
	VariantTitle   *string
}

// Cart is the subset of cart-service's response order-service needs.
type Cart struct {
	ID            uuid.UUID
	Currency      string
	Lines         []Line
	SubtotalCents int64
}

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

// Get returns the shopper's active cart. rawToken is the caller's access token.
func (c *Client) Get(ctx context.Context, rawToken string, tenantID uuid.UUID) (*Cart, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/cart", c.baseURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data struct {
			ID            uuid.UUID `json:"id"`
			Currency      string    `json:"currency"`
			SubtotalCents int64     `json:"subtotal_cents"`
			Items         []struct {
				ProductID      uuid.UUID `json:"product_id"`
				SKU            string    `json:"sku"`
				Quantity       int       `json:"quantity"`
				UnitPriceCents int64     `json:"unit_price_cents"`
				Currency       string    `json:"currency"`
				ProductName    string    `json:"product_name"`
				VariantTitle   *string   `json:"variant_title"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	if len(body.Data.Items) == 0 {
		return nil, ErrEmptyCart
	}

	lines := make([]Line, len(body.Data.Items))
	for i, it := range body.Data.Items {
		lines[i] = Line{
			ProductID:      it.ProductID,
			SKU:            it.SKU,
			Quantity:       it.Quantity,
			UnitPriceCents: it.UnitPriceCents,
			Currency:       it.Currency,
			ProductName:    it.ProductName,
			VariantTitle:   it.VariantTitle,
		}
	}
	return &Cart{
		ID:            body.Data.ID,
		Currency:      body.Data.Currency,
		Lines:         lines,
		SubtotalCents: body.Data.SubtotalCents,
	}, nil
}

// Clear empties the shopper's cart. Best-effort: the order already exists, so a
// failure here is logged by the caller, not fatal.
func (c *Client) Clear(ctx context.Context, rawToken string, tenantID uuid.UUID) error {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/cart", c.baseURL, tenantID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Authorization", "Bearer "+rawToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}
	return nil
}
