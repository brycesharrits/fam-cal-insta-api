package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/storage"
)

type UploadHandler struct {
	storage storage.ObjectStorage
}

func NewUploadHandler(storage storage.ObjectStorage) *UploadHandler {
	return &UploadHandler{storage: storage}
}

type presignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ProjectID   string `json:"project_id"`
	Month       int    `json:"month"`
}

type presignResponse struct {
	UploadURL string `json:"upload_url"`
	ObjectKey string `json:"object_key"`
	ExpiresAt string `json:"expires_at"`
}

// POST /api/v1/uploads/presign
func (h *UploadHandler) Presign(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" || req.Month == 0 {
		writeError(w, http.StatusBadRequest, "project_id and month are required")
		return
	}

	key := fmt.Sprintf("uploads/%s/%s/month_%02d_%d.jpg",
		userID, req.ProjectID, req.Month, time.Now().UnixMilli())

	ttl := 15 * time.Minute
	uploadURL, err := h.storage.GetPresignedUploadURL(r.Context(), key, ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate upload URL")
		return
	}

	writeJSON(w, http.StatusOK, presignResponse{
		UploadURL: uploadURL,
		ObjectKey: key,
		ExpiresAt: time.Now().Add(ttl).Format(time.RFC3339),
	})
}
