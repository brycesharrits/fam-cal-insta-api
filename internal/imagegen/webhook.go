package imagegen

import (
	"net/http"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

// ProviderEvent is the provider-agnostic representation of a webhook event.
// Each WebhookAdapter parses its provider's raw payload into this shape;
// downstream handlers never see provider-specific fields.
type ProviderEvent struct {
	ProviderJobID string
	Status        domain.JobStatus
	OutputURLs    []string
	ErrorMessage  string
}

// WebhookAdapter knows how to verify and parse webhook callbacks from a
// specific image generation provider. Implementations live alongside the
// provider client (e.g., internal/imagegen/replicate).
type WebhookAdapter interface {
	// Name returns the provider identifier matching ImageGenerationProvider.ProviderName().
	Name() string

	// VerifySignature authenticates the request against the configured secret.
	// Returns nil if valid; an error otherwise. Implementations should consume
	// the request body via r.Body and leave it ready for subsequent reads
	// (typically by replacing r.Body with a fresh io.NopCloser).
	VerifySignature(r *http.Request, secret string) error

	// ParseEvent translates the raw webhook body into the normalized event.
	ParseEvent(body []byte) (*ProviderEvent, error)
}
