// Package catalog is a thin client for product-service. cart-service uses it to
// snapshot a variant's price and names when a line is added.
package catalog

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
	// ErrNotAvailable means the product is not on sale (not found or not ACTIVE),
	// or has no variant with the given sku.
	ErrNotAvailable = errors.New("variant is not available in the catalog")
	// ErrUpstream means product-service was unreachable or errored.
	ErrUpstream = errors.New("product-service is unavailable")
)

// Variant is the slice of product-service data a cart line snapshots.
type Variant struct {
	ProductID    uuid.UUID
	SKU          string
	PriceCents   int64
	Currency     string
	ProductName  string
	VariantTitle *string
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

// Lookup fetches product productID in tenantID and returns the variant whose sku
// matches (case-insensitive). rawToken is forwarded though the endpoint is public.
func (c *Client) Lookup(ctx context.Context, rawToken string, tenantID, productID uuid.UUID, sku string) (*Variant, error) {
	url := fmt.Sprintf("%s/api/v1/tenants/%s/catalog/%s", c.baseURL, tenantID, productID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if rawToken != "" {
		req.Header.Set("Authorization", "Bearer "+rawToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return nil, ErrNotAvailable
	default:
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var body struct {
		Data struct {
			Name     string `json:"name"`
			Variants []struct {
				ID         uuid.UUID `json:"id"`
				ProductID  uuid.UUID `json:"product_id"`
				SKU        string    `json:"sku"`
				Title      *string   `json:"title"`
				PriceCents int64     `json:"price_cents"`
				Currency   string    `json:"currency"`
			} `json:"variants"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}

	for _, v := range body.Data.Variants {
		if strings.EqualFold(v.SKU, sku) {
			return &Variant{
				ProductID:    productID,
				SKU:          v.SKU,
				PriceCents:   v.PriceCents,
				Currency:     v.Currency,
				ProductName:  body.Data.Name,
				VariantTitle: v.Title,
			}, nil
		}
	}
	return nil, ErrNotAvailable
}
