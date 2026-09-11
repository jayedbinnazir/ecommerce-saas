// Package paymentclient calls payment-service's internal API. It authenticates
// with a shared X-Internal-Key, not a user token.
package paymentclient

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
	// ErrRejected means payment-service refused the request (bad method/amount,
	// nothing to settle/refund). The wrapped message explains why.
	ErrRejected = errors.New("payment-service rejected the request")
	// ErrUpstream means payment-service was unreachable or errored.
	ErrUpstream = errors.New("payment-service is unavailable")
)

// Payment is the slice of payment-service's response order-service needs.
type Payment struct {
	ID           uuid.UUID
	Method       string
	Status       string  // PENDING | CAPTURED | FAILED | REFUNDED
	ClientSecret *string // CARD only, on create
}

// CreateRequest mirrors payment-service's internal create body (tenant is in the URL).
type CreateRequest struct {
	OrderID     uuid.UUID `json:"order_id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	Method      string    `json:"method"`
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
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*Payment, error) {
	body, _ := json.Marshal(req)
	return c.call(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/internal/tenants/%s/payments", tenantID), body)
}

func (c *Client) Sync(ctx context.Context, tenantID, paymentID uuid.UUID) (*Payment, error) {
	return c.call(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/internal/tenants/%s/payments/%s/sync", tenantID, paymentID), nil)
}

func (c *Client) Settle(ctx context.Context, tenantID, paymentID uuid.UUID) (*Payment, error) {
	return c.call(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/internal/tenants/%s/payments/%s/settle", tenantID, paymentID), nil)
}

// Cancel voids a payment that hasn't captured yet — called when an order is
// cancelled while still PENDING_PAYMENT.
func (c *Client) Cancel(ctx context.Context, tenantID, paymentID uuid.UUID) (*Payment, error) {
	return c.call(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/internal/tenants/%s/payments/%s/cancel", tenantID, paymentID), nil)
}

// Refund reverses a payment. amountCents nil = full refund; a value = partial.
func (c *Client) Refund(ctx context.Context, tenantID, paymentID uuid.UUID, amountCents *int64) (*Payment, error) {
	var body []byte
	if amountCents != nil {
		body, _ = json.Marshal(map[string]int64{"amount_cents": *amountCents})
	}
	return c.call(ctx, http.MethodPost,
		fmt.Sprintf("/api/v1/internal/tenants/%s/payments/%s/refund", tenantID, paymentID), body)
}

func (c *Client) call(ctx context.Context, method, path string, body []byte) (*Payment, error) {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode < 300:
		// ok
	case resp.StatusCode == http.StatusUnprocessableEntity ||
		resp.StatusCode == http.StatusConflict ||
		resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: %s", ErrRejected, errorMessage(resp))
	default:
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var out struct {
		Data struct {
			ID           uuid.UUID `json:"id"`
			Method       string    `json:"method"`
			Status       string    `json:"status"`
			ClientSecret *string   `json:"client_secret"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	return &Payment{
		ID:           out.Data.ID,
		Method:       out.Data.Method,
		Status:       out.Data.Status,
		ClientSecret: out.Data.ClientSecret,
	}, nil
}

func errorMessage(resp *http.Response) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Error.Message == "" {
		return "payment rejected"
	}
	return body.Error.Message
}
