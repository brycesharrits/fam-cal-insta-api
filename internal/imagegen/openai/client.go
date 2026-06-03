// Package openai is a thin helper for hitting OpenAI's gpt-image-1 endpoints
// directly from the Test Lab handler. It does NOT implement the
// ImageGenerationProvider interface — Test Lab is intentionally a disposable
// surface that bypasses the normal job/worker pipeline. Delete this package
// when the spike concludes and we commit to a single provider.
package openai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

const (
	generationsURL = "https://api.openai.com/v1/images/generations"
	editsURL       = "https://api.openai.com/v1/images/edits"
	model          = "gpt-image-1"
	defaultSize    = "1024x1024"
	defaultQuality = "medium"
)

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 5 * time.Minute},
	}
}

// Result is the decoded image bytes + the MIME type returned by OpenAI.
type Result struct {
	Bytes []byte
	Mime  string
}

type imageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// TextToImage calls /v1/images/generations with a prompt.
func (c *Client) TextToImage(prompt string) (*Result, error) {
	body, err := json.Marshal(map[string]any{
		"model":         model,
		"prompt":        prompt,
		"n":             1,
		"size":          defaultSize,
		"quality":       defaultQuality,
		"output_format": "png",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, generationsURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return c.do(req, "image/png")
}

// Edit calls /v1/images/edits with a reference image plus a prompt.
// input_fidelity=low means the photo informs mood/style rather than
// preserving exact structure — matches the product's Option A.
func (c *Client) Edit(prompt string, imageBytes []byte, imageMime string) (*Result, error) {
	ext := extFromMime(imageMime)
	if ext == "" {
		return nil, fmt.Errorf("unsupported image mime type %q (use image/jpeg, image/png, or image/webp)", imageMime)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("model", model)
	_ = w.WriteField("prompt", prompt)
	_ = w.WriteField("n", "1")
	_ = w.WriteField("size", defaultSize)
	_ = w.WriteField("quality", defaultQuality)
	_ = w.WriteField("input_fidelity", "low")
	_ = w.WriteField("output_format", "png")

	// CreateFormFile hardcodes application/octet-stream, which OpenAI rejects.
	// Build the part header manually with the real image MIME type.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename=%q`, "input"+ext))
	h.Set("Content-Type", imageMime)
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("create form part: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("write image: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, editsURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return c.do(req, "image/png")
}

func (c *Client) do(req *http.Request, outMime string) (*Result, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		_ = json.Unmarshal(raw, &apiErr)
		if apiErr.Error.Message != "" {
			return nil, fmt.Errorf("openai %d: %s (%s)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Code)
		}
		return nil, fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
	}

	var out imageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("openai returned no image data")
	}
	img, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	return &Result{Bytes: img, Mime: outMime}, nil
}

func extFromMime(m string) string {
	switch strings.ToLower(m) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

