package replicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/imagegen"
)

const replicateAPIBase = "https://api.replicate.com/v1"

type FluxProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewFluxProvider(apiKey, model string) *FluxProvider {
	return &FluxProvider{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (f *FluxProvider) ProviderName() string {
	return "replicate/flux"
}

// Generate is synchronous — polls until complete. Use in dev/testing.
func (f *FluxProvider) Generate(ctx context.Context, req imagegen.GenerationRequest) (*imagegen.GenerationResult, error) {
	predID, err := f.GenerateAsync(ctx, req)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		result, done, err := f.PollStatus(ctx, predID)
		if err != nil {
			return nil, err
		}
		if done {
			return result, nil
		}
	}
}

// GenerateAsync submits the prediction to Replicate and returns the prediction ID.
func (f *FluxProvider) GenerateAsync(ctx context.Context, req imagegen.GenerationRequest) (string, error) {
	width := req.Width
	height := req.Height
	if width == 0 {
		width = 2400 // calendar landscape default
	}
	if height == 0 {
		height = 1800
	}

	input := map[string]interface{}{
		"prompt":        req.Prompt,
		"width":         width,
		"height":        height,
		"output_format": "jpg",
		"output_quality": 95,
	}

	// Include reference image if provided
	if req.ReferenceImageURL != "" {
		input["image"] = req.ReferenceImageURL
		input["image_strength"] = 0.35 // how much to deviate from reference
	}

	body := map[string]interface{}{
		"version": f.model,
		"input":   input,
	}

	// Set webhook URL if job ID is provided — we use the job ID as a correlation key
	// The webhook URL must be configured at the service level and injected separately.
	// For now, we use polling as the primary mechanism.

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/predictions", replicateAPIBase), bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Token "+f.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("replicate API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("replicate API error %d: %s", resp.StatusCode, string(body))
	}

	var prediction struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return "", fmt.Errorf("decoding prediction response: %w", err)
	}

	return prediction.ID, nil
}

// PollStatus checks the status of a Replicate prediction.
func (f *FluxProvider) PollStatus(ctx context.Context, predictionID string) (*imagegen.GenerationResult, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/predictions/%s", replicateAPIBase, predictionID), nil)
	if err != nil {
		return nil, false, err
	}
	httpReq.Header.Set("Authorization", "Token "+f.apiKey)

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, fmt.Errorf("polling prediction: %w", err)
	}
	defer resp.Body.Close()

	var prediction struct {
		ID     string   `json:"id"`
		Status string   `json:"status"` // starting | processing | succeeded | failed | canceled
		Output []string `json:"output"` // Flux returns array of image URLs
		Error  string   `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return nil, false, err
	}

	switch prediction.Status {
	case "succeeded":
		if len(prediction.Output) == 0 {
			return nil, false, fmt.Errorf("prediction succeeded but no output")
		}
		return &imagegen.GenerationResult{
			ImageURL:  prediction.Output[0],
			Provider:  f.ProviderName(),
			ModelID:   f.model,
			CreatedAt: time.Now(),
		}, true, nil
	case "failed", "canceled":
		return nil, false, fmt.Errorf("prediction %s: %s", prediction.Status, prediction.Error)
	default:
		// starting or processing — not done yet
		return nil, false, nil
	}
}
