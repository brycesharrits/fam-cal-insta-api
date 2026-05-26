// Throwaway PoC: hits OpenAI's gpt-image-1 image-edit endpoint with a
// user-provided reference photo plus a prompt, saves the result to spike-out/.
// Tests the "real family photo influences the output" path. Not wired into
// the main server; delete once we've committed to a provider.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const apiURL = "https://api.openai.com/v1/images/edits"

const defaultPrompt = `Transform this photo into a whimsical, goofy holiday scene
for January in a family wall calendar: oversized novelty New Year's party hats,
confetti drifting through the air, exaggerated celebratory expressions,
warm soft lighting, slightly cartoonish illustration style with a playful tone.`

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
	imagePath := flag.String("image", "", "path to reference image (png/jpeg/webp, <50MB) — required")
	prompt := flag.String("prompt", defaultPrompt, "transformation prompt")
	size := flag.String("size", "1024x1024", "output size: 1024x1024, 1536x1024, 1024x1536, auto")
	quality := flag.String("quality", "medium", "quality: low, medium, high, auto")
	fidelity := flag.String("input-fidelity", "low", "input_fidelity: low = photo as mood/style, high = preserve photo structure")
	label := flag.String("label", "edit-goofy-january", "filename label")
	flag.Parse()

	if *imagePath == "" {
		log.Fatal("-image flag is required (path to reference photo)")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY env var is required")
	}

	file, err := os.Open(*imagePath)
	if err != nil {
		log.Fatalf("open image: %v", err)
	}
	defer file.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("model", "gpt-image-1")
	_ = w.WriteField("prompt", *prompt)
	_ = w.WriteField("n", "1")
	_ = w.WriteField("size", *size)
	_ = w.WriteField("quality", *quality)
	_ = w.WriteField("input_fidelity", *fidelity)
	_ = w.WriteField("output_format", "png")

	// CreateFormFile would hardcode application/octet-stream, which OpenAI rejects.
	// Build the part header manually with the real image MIME type.
	mimeType := mimeTypeForExt(filepath.Ext(*imagePath))
	if mimeType == "" {
		log.Fatalf("unsupported image extension %q (use .jpg/.jpeg/.png/.webp)", filepath.Ext(*imagePath))
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename=%q`, filepath.Base(*imagePath)))
	h.Set("Content-Type", mimeType)
	part, err := w.CreatePart(h)
	if err != nil {
		log.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		log.Fatalf("copy image: %v", err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("close multipart: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, &body)
	if err != nil {
		log.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	log.Printf("calling OpenAI edit: model=gpt-image-1 image=%s size=%s quality=%s input_fidelity=%s prompt_chars=%d",
		filepath.Base(*imagePath), *size, *quality, *fidelity, len(*prompt))
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

func mimeTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}
