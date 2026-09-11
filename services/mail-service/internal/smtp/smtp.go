// Package smtp sends an email over a relay. With no host configured it runs in
// stub mode: the message is logged and reported as sent, so local dev needs no
// mail server.
package smtp

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"strings"
)

// Sender delivers one rendered email.
type Sender interface {
	// Live reports whether a real relay is configured (vs the stub).
	Live() bool
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// New returns a real SMTP sender when host is set, otherwise the stub.
func New(host, port, username, password, from string) Sender {
	if strings.TrimSpace(host) == "" {
		return stub{from: orDefault(from, "no-reply@localhost")}
	}
	// Only authenticate when a username is configured. With no credentials we
	// must pass a nil auth so net/smtp does not attempt (and reject) PLAIN over
	// a plaintext link — this is the local-dev / Mailpit case.
	var auth smtp.Auth
	if strings.TrimSpace(username) != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	fromHdr := orDefault(from, username)
	return &relay{
		addr:     host + ":" + orDefault(port, "587"),
		auth:     auth,
		fromHdr:  fromHdr,
		fromAddr: orDefault(envelopeAddr(fromHdr), "no-reply@localhost"),
	}
}

// ---- stub ----

type stub struct{ from string }

func (stub) Live() bool { return false }

func (s stub) Send(_ context.Context, to, subject, htmlBody string) error {
	log.Printf("[mail:stub] from=%s to=%s subject=%q (%d bytes body) — not actually sent",
		s.from, to, subject, len(htmlBody))
	return nil
}

// ---- relay ----

type relay struct {
	addr     string
	auth     smtp.Auth
	fromHdr  string // full "Name <addr>" for the From: header
	fromAddr string // bare address for the SMTP MAIL FROM envelope
}

func (r *relay) Live() bool { return true }

func (r *relay) Send(_ context.Context, to, subject, htmlBody string) error {
	msg := "From: " + r.fromHdr + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + htmlBody

	if err := smtp.SendMail(r.addr, r.auth, r.fromAddr, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// envelopeAddr pulls the bare "user@host" out of a "Name <user@host>" string;
// the SMTP MAIL FROM command rejects the display-name form.
func envelopeAddr(from string) string {
	if a, err := mail.ParseAddress(from); err == nil {
		return a.Address
	}
	return from
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
