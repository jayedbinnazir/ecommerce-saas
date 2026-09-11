// Package mailclient calls mail-service's internal send API. Sending mail is
// best-effort from a notification's point of view — a failure is logged, not
// propagated.
package mailclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

// New returns a client. An empty baseURL yields a no-op client (mail disabled).
func New(baseURL, internalKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     internalKey,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled reports whether a mail-service URL is configured.
func (c *Client) Enabled() bool { return c.baseURL != "" }

// Send renders template with data and delivers it to addr.
func (c *Client) Send(ctx context.Context, addr, template string, data map[string]any) error {
	if !c.Enabled() {
		return nil
	}

	payload, _ := json.Marshal(struct {
		To       string         `json:"to"`
		Template string         `json:"template"`
		Data     map[string]any `json:"data"`
	}{To: addr, Template: template, Data: data})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/mail", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mail-service returned status %d", resp.StatusCode)
	}
	return nil
}
