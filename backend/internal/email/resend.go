// Package email contains the outbound email adapter used by account recovery.
// Provider credentials stay in environment configuration and are never logged.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

// Sender is the small boundary the service uses so tests can substitute a
// fake without making network calls.
type Sender interface {
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

type Resend struct {
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

func NewResend(apiKey, from string, client *http.Client) *Resend {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Resend{
		apiKey:   strings.TrimSpace(apiKey),
		from:     strings.TrimSpace(from),
		endpoint: "https://api.resend.com/emails",
		client:   client,
	}
}

func (r *Resend) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	if r == nil || r.apiKey == "" || r.from == "" {
		return fmt.Errorf("resend is not configured")
	}
	link := html.EscapeString(resetURL)
	payload := map[string]any{
		"from":    r.from,
		"to":      []string{to},
		"subject": "Atur ulang kata sandi EFA",
		"text":    "Kami menerima permintaan mengatur ulang kata sandi EFA. Buka tautan ini jika kamu memintanya: " + resetURL + "\n\nTautan berlaku terbatas dan hanya dapat digunakan sekali. Abaikan email ini jika kamu tidak meminta reset.",
		"html":    "<p>Kami menerima permintaan mengatur ulang kata sandi <strong>EFA</strong>.</p><p><a href=\"" + link + "\">Atur ulang kata sandi</a></p><p>Tautan berlaku terbatas dan hanya dapat digunakan sekali. Abaikan email ini jika kamu tidak meminta reset.</p>",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode resend request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("resend returned status %d", resp.StatusCode)
	}
	return nil
}
