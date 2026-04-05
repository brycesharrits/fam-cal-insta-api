package imagegen

import (
	"context"
	"time"
)

// GenerationRequest holds all inputs needed to generate one calendar month image.
type GenerationRequest struct {
	// ReferenceImageURL is a publicly accessible URL to the reference photo.
	// Used to inform the style/mood/season of the generated image.
	ReferenceImageURL string

	// Prompt is the fully built text prompt (constructed by PromptBuilder).
	Prompt string

	Width  int
	Height int

	// JobID is the internal generation_jobs.id — passed through for correlation.
	JobID string
}

// GenerationResult is returned when a generation completes successfully.
type GenerationResult struct {
	ImageURL  string
	Provider  string
	ModelID   string
	CreatedAt time.Time
}

// ImageGenerationProvider is the interface all AI image providers must implement.
// Swap Flux for any other provider by implementing this interface.
type ImageGenerationProvider interface {
	// Generate is synchronous — blocks until the image is ready.
	// Use for development/testing; prefer GenerateAsync in production.
	Generate(ctx context.Context, req GenerationRequest) (*GenerationResult, error)

	// GenerateAsync submits the job and returns immediately with a provider-specific
	// prediction ID. The result is delivered via webhook or polling.
	GenerateAsync(ctx context.Context, req GenerationRequest) (predictionID string, err error)

	// PollStatus checks the status of an async job.
	// Returns (result, true, nil) when complete.
	// Returns (nil, false, nil) when still in progress.
	// Returns (nil, false, err) on failure.
	PollStatus(ctx context.Context, predictionID string) (*GenerationResult, bool, error)

	// ProviderName returns a human-readable name for logging/metrics.
	ProviderName() string
}
