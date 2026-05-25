// Throwaway PoC: hits OpenAI's gpt-image-1 text-to-image endpoint with a
// hardcoded family-calendar prompt and saves the result to spike-out/.
// Not wired into the main server. Delete this whole directory once we've
// committed to a provider and built the production integration.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const apiURL = "https://api.openai.com/v1/images/generations"

const defaultPrompt = `A whimsical, goofy holiday scene for January, suitable for a family wall calendar:
a family gathered in a cozy living room, oversized novelty New Year's party hats,
confetti drifting through the air, exaggerated celebratory expressions,
warm soft lighting, slightly cartoonish illustration style with a playful tone,
high quality, painterly detail.`

type imageRequest struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	N            int    `json:"n"`
	Size         string `json:"size"`
	Quality      string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

type imageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func main() {
	prompt := flag.String("prompt", defaultPrompt, "image prompt")
	size := flag.String("size", "1024x1024", "image size: 1024x1024, 1536x1024, 1024x1536, auto")
	quality := flag.String("quality", "medium", "quality: low, medium, high, auto")
	label := flag.String("label", "goofy-january", "filename label")
	flag.Parse()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY env var is required")
	}

	body, err := json.Marshal(imageRequest{
		Model:        "gpt-image-1",
		Prompt:       *prompt,
		N:            1,
		Size:         *size,
		Quality:      *quality,
		OutputFormat: "png",
	})
	if err != nil {
		log.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("calling OpenAI: model=gpt-image-1 size=%s quality=%s prompt_chars=%d", *size, *quality, len(*prompt))
	start := time.Now()

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("http error: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		_ = json.Unmarshal(raw, &apiErr)
		log.Fatalf("openai %d: %s (%s / %s)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Type, apiErr.Error.Code)
	}

	var out imageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatalf("decode response: %v", err)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		log.Fatal("no image data returned")
	}

	img, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
	if err != nil {
		log.Fatalf("decode base64: %v", err)
	}

	outDir := "spike-out"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	filename := fmt.Sprintf("%s-%s.png", time.Now().Format("20060102-150405"), *label)
	path := filepath.Join(outDir, filename)
	if err := os.WriteFile(path, img, 0o644); err != nil {
		log.Fatalf("write file: %v", err)
	}

	log.Printf("done in %s -> %s (%d KB)", elapsed.Round(time.Millisecond), path, len(img)/1024)
	if out.Usage != nil {
		log.Printf("usage: input=%d output=%d total=%d tokens",
			out.Usage.InputTokens, out.Usage.OutputTokens, out.Usage.TotalTokens)
	}
}
