// Package gateway is a tiny Stripe client for card payments. It speaks the Stripe
// REST API directly (form-encoded, Bearer secret key) so we don't pull in the
// full SDK. When no secret key is configured it falls back to a stub that
// "succeeds" instantly, so local dev works without a Stripe account.
package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status is our normalised view of a Stripe PaymentIntent status.
type Status string

const (
	StatusPending   Status = "PENDING"   // needs the customer to complete payment
	StatusSucceeded Status = "SUCCEEDED" // money captured
	StatusFailed    Status = "FAILED"    // canceled / unrecoverable
)

// Intent is the slice of a Stripe PaymentIntent we care about.
type Intent struct {
	ID           string
	ClientSecret string
	Status       Status
}

var ErrWebhookUnsupported = errors.New("webhook verification needs a configured webhook secret")

// Gateway is the card-payment provider.
type Gateway interface {
	// Live reports whether a real Stripe key is configured (vs the stub).
	Live() bool
	CreateIntent(ctx context.Context, amountCents int64, currency, orderID string) (*Intent, error)
	GetIntent(ctx context.Context, id string) (*Intent, error)
	// Refund reverses a payment. amountCents nil = full refund.
	Refund(ctx context.Context, intentID string, amountCents *int64) error
	// Cancel voids a PaymentIntent that hasn't captured yet (e.g. the order was
	// cancelled while still PENDING_PAYMENT). Stripe rejects this if the intent
	// already succeeded — the caller should treat that as "too late" rather
	// than force local state to match, and let the normal capture webhook
	// reconcile it instead.
	Cancel(ctx context.Context, intentID string) error
	// VerifyWebhook checks the Stripe-Signature header and returns the event type
	// and the PaymentIntent id the event concerns.
	VerifyWebhook(payload []byte, signatureHeader string) (eventType, intentID string, err error)
}

// New returns a real Stripe gateway when secretKey is set, otherwise the stub.
func New(secretKey, webhookSecret string) Gateway {
	if strings.TrimSpace(secretKey) == "" {
		return stub{}
	}
	return &stripe{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

// ---------------------------------------------------------------------
// stub — no Stripe account needed
// ---------------------------------------------------------------------

type stub struct{}

func (stub) Live() bool { return false }

func (stub) CreateIntent(_ context.Context, _ int64, _, _ string) (*Intent, error) {
	id := "pi_stub_" + uuid.NewString()
	return &Intent{ID: id, ClientSecret: id + "_secret_stub", Status: StatusSucceeded}, nil
}

func (stub) GetIntent(_ context.Context, id string) (*Intent, error) {
	return &Intent{ID: id, ClientSecret: id + "_secret_stub", Status: StatusSucceeded}, nil
}

func (stub) Refund(_ context.Context, _ string, _ *int64) error { return nil }

func (stub) Cancel(_ context.Context, _ string) error { return nil }

func (stub) VerifyWebhook([]byte, string) (string, string, error) {
	return "", "", ErrWebhookUnsupported
}

// ---------------------------------------------------------------------
// stripe — real REST calls
// ---------------------------------------------------------------------

const stripeAPI = "https://api.stripe.com/v1"

type stripe struct {
	secretKey     string
	webhookSecret string
	http          *http.Client
}

func (s *stripe) Live() bool { return true }

func (s *stripe) CreateIntent(ctx context.Context, amountCents int64, currency, orderID string) (*Intent, error) {
	form := url.Values{}
	form.Set("amount", strconv.FormatInt(amountCents, 10))
	form.Set("currency", strings.ToLower(currency))
	form.Set("automatic_payment_methods[enabled]", "true")
	form.Set("metadata[order_id]", orderID)

	var pi stripeIntent
	if err := s.do(ctx, http.MethodPost, "/payment_intents", form, &pi); err != nil {
		return nil, err
	}
	return pi.toIntent(), nil
}

func (s *stripe) GetIntent(ctx context.Context, id string) (*Intent, error) {
	var pi stripeIntent
	if err := s.do(ctx, http.MethodGet, "/payment_intents/"+url.PathEscape(id), nil, &pi); err != nil {
		return nil, err
	}
	return pi.toIntent(), nil
}

func (s *stripe) Refund(ctx context.Context, intentID string, amountCents *int64) error {
	form := url.Values{}
	form.Set("payment_intent", intentID)
	if amountCents != nil {
		form.Set("amount", strconv.FormatInt(*amountCents, 10))
	}
	return s.do(ctx, http.MethodPost, "/refunds", form, nil)
}

func (s *stripe) Cancel(ctx context.Context, intentID string) error {
	return s.do(ctx, http.MethodPost, "/payment_intents/"+url.PathEscape(intentID)+"/cancel", url.Values{}, nil)
}

func (s *stripe) VerifyWebhook(payload []byte, signatureHeader string) (string, string, error) {
	if s.webhookSecret == "" {
		return "", "", ErrWebhookUnsupported
	}

	var timestamp, sig string
	for _, part := range strings.Split(signatureHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	if timestamp == "" || sig == "" {
		return "", "", errors.New("malformed Stripe-Signature header")
	}

	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", "", errors.New("webhook signature mismatch")
	}

	var event struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return "", "", err
	}
	return event.Type, event.Data.Object.ID, nil
}

func (s *stripe) do(ctx context.Context, method, path string, form url.Values, out any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, stripeAPI+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("stripe %s %s: status %d: %s", method, path, resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

type stripeIntent struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Status       string `json:"status"`
}

func (pi stripeIntent) toIntent() *Intent {
	status := StatusPending
	switch pi.Status {
	case "succeeded":
		status = StatusSucceeded
	case "canceled":
		status = StatusFailed
	}
	return &Intent{ID: pi.ID, ClientSecret: pi.ClientSecret, Status: status}
}
