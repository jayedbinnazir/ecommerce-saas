// Package templates renders the built-in transactional emails. Each template is
// a subject line plus an html/template body; callers pass a free-form data map.
package templates

import (
	"bytes"
	"html/template"

	"github.com/jayedbinnazir/mail-service/internal/mail/domain"
)

// Rendered is a ready-to-send email.
type Rendered struct {
	Subject string
	HTML    string
}

type tmpl struct {
	subject string
	body    *template.Template
}

var registry = map[string]tmpl{
	"welcome": {
		subject: "Welcome to {{.store_name}}",
		body: must(`<h2>Welcome, {{.name}}!</h2>
<p>Your account is ready. You can now create a store and start selling.</p>`),
	},
	"subscription_started": {
		subject: "Your {{.plan_name}} subscription is active",
		body: must(`<h2>Subscription active</h2>
<p>Hi {{.name}}, your <strong>{{.plan_name}}</strong> plan is active until {{.period_end}}.</p>`),
	},
	"order_confirmation": {
		subject: "Order {{.order_number}} received",
		body: must(`<h2>Thanks for your order!</h2>
<p>Order <strong>{{.order_number}}</strong> — total {{.total}}.</p>
<p>Payment method: {{.payment_method}}. We'll email you when it ships.</p>`),
	},
	"order_shipped": {
		subject: "Order {{.order_number}} is on its way",
		body: must(`<h2>Your order shipped</h2>
<p>Order <strong>{{.order_number}}</strong> has been fulfilled and is on its way.</p>`),
	},
	"order_cancelled": {
		subject: "Order {{.order_number}} was cancelled",
		body: must(`<h2>Order cancelled</h2>
<p>Order <strong>{{.order_number}}</strong> has been cancelled{{if .refunded}} and your payment refunded{{end}}.</p>`),
	},
	"payment_receipt": {
		subject: "Receipt for order {{.order_number}}",
		body: must(`<h2>Payment received</h2>
<p>We received {{.amount}} for order <strong>{{.order_number}}</strong>.</p>`),
	},
}

// Names lists the available template ids.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	return out
}

// Render fills a template with data. Missing keys render as "<no value>".
func Render(name string, data map[string]any) (*Rendered, error) {
	t, ok := registry[name]
	if !ok {
		return nil, domain.ErrUnknownTemplate
	}

	subject, err := renderString(t.subject, data)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.body.Execute(&buf, data); err != nil {
		return nil, err
	}
	return &Rendered{Subject: subject, HTML: layout(buf.String())}, nil
}

func renderString(s string, data map[string]any) (string, error) {
	t, err := template.New("subject").Parse(s)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func layout(inner string) string {
	return `<!doctype html><html><body style="font-family:system-ui,sans-serif;color:#222">` +
		inner +
		`<hr><p style="color:#888;font-size:12px">Sent by the platform — do not reply.</p></body></html>`
}

func must(body string) *template.Template {
	return template.Must(template.New("body").Parse(body))
}
