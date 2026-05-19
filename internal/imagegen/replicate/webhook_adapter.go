package replicate

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen"
)

// signatureTolerance is the maximum clock skew accepted between the timestamp
// in the signed payload and the server's clock.
const signatureTolerance = 5 * time.Minute

// WebhookAdapter implements imagegen.WebhookAdapter for Replicate.
// Signature verification follows the Standard Webhooks spec, which Replicate
// adopts: HMAC-SHA256 over "{webhook-id}.{webhook-timestamp}.{body}".
type WebhookAdapter struct{}

func NewWebhookAdapter() *WebhookAdapter { return &WebhookAdapter{} }

func (a *WebhookAdapter) Name() string { return "replicate/flux" }

func (a *WebhookAdapter) VerifySignature(r *http.Request, secret string) error {
	if secret == "" {
		return errors.New("webhook secret not configured")
	}

	webhookID := r.Header.Get("webhook-id")
	timestamp := r.Header.Get("webhook-timestamp")
	sigHeader := r.Header.Get("webhook-signature")
	if webhookID == "" || timestamp == "" || sigHeader == "" {
		return errors.New("missing required webhook signature headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook-timestamp: %w", err)
	}
	delta := time.Since(time.Unix(ts, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > signatureTolerance {
		return fmt.Errorf("webhook timestamp outside tolerance window")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading webhook body: %w", err)
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	key, err := decodeSecret(secret)
	if err != nil {
		return fmt.Errorf("decoding webhook secret: %w", err)
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(webhookID))
	mac.Write([]byte("."))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := mac.Sum(nil)

	for _, part := range strings.Fields(sigHeader) {
		version, encoded, ok := strings.Cut(part, ",")
		if !ok || version != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		if hmac.Equal(got, expected) {
			return nil
		}
	}
	return errors.New("no valid signature found")
}

func (a *WebhookAdapter) ParseEvent(body []byte) (*imagegen.ProviderEvent, error) {
	var payload struct {
		ID     string   `json:"id"`
		Status string   `json:"status"`
		Output []string `json:"output"`
		Error  string   `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decoding replicate payload: %w", err)
	}

	evt := &imagegen.ProviderEvent{
		ProviderJobID: payload.ID,
		OutputURLs:    payload.Output,
		ErrorMessage:  payload.Error,
	}
	switch payload.Status {
	case "succeeded":
		evt.Status = domain.JobStatusComplete
	case "failed", "canceled":
		evt.Status = domain.JobStatusFailed
	case "starting", "processing":
		evt.Status = domain.JobStatusProcessing
	default:
		evt.Status = domain.JobStatusQueued
	}
	return evt, nil
}

// decodeSecret accepts the Standard Webhooks "whsec_<base64>" form as well as
// a raw base64 string. Falls back to using the raw bytes if no base64 prefix
// pattern matches, to support tooling that hands the key over unencoded.
func decodeSecret(secret string) ([]byte, error) {
	trimmed := strings.TrimPrefix(secret, "whsec_")
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	return []byte(secret), nil
}
