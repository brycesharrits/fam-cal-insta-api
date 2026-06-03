package v1

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen/openai"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

// TestGenHandler powers the disposable Test Lab medium. It bypasses the
// normal project/job pipeline and calls OpenAI inline. Delete this handler
// when the spike concludes.
type TestGenHandler struct {
	repo   repository.TestGenerationRepository
	openai *openai.Client
}

func NewTestGenHandler(repo repository.TestGenerationRepository, client *openai.Client) *TestGenHandler {
	return &TestGenHandler{repo: repo, openai: client}
}

type testGenerateRequest struct {
	Mode             string `json:"mode"`
	Prompt           string `json:"prompt"`
	InputImageBase64 string `json:"input_image_base64,omitempty"`
	InputImageMime   string `json:"input_image_mime,omitempty"`
}

type testGenerateResponse struct {
	ID                string `json:"id"`
	Mode              string `json:"mode"`
	Prompt            string `json:"prompt"`
	Status            string `json:"status"`
	OutputImageBase64 string `json:"output_image_base64,omitempty"`
	OutputImageMime   string `json:"output_image_mime,omitempty"`
	ErrorMessage      string `json:"error_message,omitempty"`
	DurationMs        int    `json:"duration_ms"`
	CreatedAt         string `json:"created_at"`
}

// POST /api/v1/test/generate
func (h *TestGenHandler) Generate(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if h.openai == nil {
		writeError(w, http.StatusServiceUnavailable, "OPENAI_API_KEY not configured on server")
		return
	}

	var req testGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	mode := domain.TestGenerationMode(req.Mode)
	if mode != domain.TestGenerationModeText && mode != domain.TestGenerationModeEdit {
		writeError(w, http.StatusBadRequest, "mode must be 'text' or 'edit'")
		return
	}

	var inputBytes []byte
	if mode == domain.TestGenerationModeEdit {
		if req.InputImageBase64 == "" || req.InputImageMime == "" {
			writeError(w, http.StatusBadRequest, "edit mode requires input_image_base64 and input_image_mime")
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(req.InputImageBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "input_image_base64 is not valid base64")
			return
		}
		inputBytes = decoded
	}

	start := time.Now()
	var (
		result   *openai.Result
		genErr   error
	)
	switch mode {
	case domain.TestGenerationModeText:
		result, genErr = h.openai.TextToImage(req.Prompt)
	case domain.TestGenerationModeEdit:
		result, genErr = h.openai.Edit(req.Prompt, inputBytes, req.InputImageMime)
	}
	elapsed := int(time.Since(start) / time.Millisecond)

	row := &domain.TestGeneration{
		UserID:          userID,
		Mode:            mode,
		Prompt:          req.Prompt,
		InputImageBytes: inputBytes,
		InputImageMime:  req.InputImageMime,
		DurationMs:      elapsed,
	}

	if genErr != nil {
		slog.Error("test_gen openai call failed", "user_id", userID, "mode", mode, "error", genErr)
		row.Status = domain.TestGenerationStatusFailed
		row.ErrorMessage = genErr.Error()
		_ = h.repo.Create(r.Context(), row)
		writeJSON(w, http.StatusBadGateway, testGenerateResponse{
			ID:           row.ID,
			Mode:         req.Mode,
			Prompt:       req.Prompt,
			Status:       string(row.Status),
			ErrorMessage: row.ErrorMessage,
			DurationMs:   elapsed,
			CreatedAt:    row.CreatedAt.Format(time.RFC3339),
		})
		return
	}

	row.OutputImageBytes = result.Bytes
	row.OutputImageMime = result.Mime
	row.Status = domain.TestGenerationStatusComplete
	if err := h.repo.Create(r.Context(), row); err != nil {
		slog.Error("test_gen save row failed", "user_id", userID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save result")
		return
	}

	writeJSON(w, http.StatusOK, testGenerateResponse{
		ID:                row.ID,
		Mode:              req.Mode,
		Prompt:            req.Prompt,
		Status:            string(row.Status),
		OutputImageBase64: base64.StdEncoding.EncodeToString(result.Bytes),
		OutputImageMime:   result.Mime,
		DurationMs:        elapsed,
		CreatedAt:         row.CreatedAt.Format(time.RFC3339),
	})
}
