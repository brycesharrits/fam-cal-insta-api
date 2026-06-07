package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen"
	"github.com/brycesharrits/fam-cal-insta/internal/jobs"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

type GenerationHandler struct {
	projectRepo     repository.ProjectRepository
	monthRepo       repository.MonthRepository
	jobRepo         repository.GenerationJobRepository
	tokenRepo       repository.TokenRepository
	genWorker       *jobs.GenerationWorker
	webhookAdapters map[string]imagegen.WebhookAdapter
	webhookSecrets  map[string]string
	fullGenCost     int
	singleGenCost   int
}

func NewGenerationHandler(
	projectRepo repository.ProjectRepository,
	monthRepo repository.MonthRepository,
	jobRepo repository.GenerationJobRepository,
	tokenRepo repository.TokenRepository,
	genWorker *jobs.GenerationWorker,
	webhookAdapters map[string]imagegen.WebhookAdapter,
	webhookSecrets map[string]string,
	fullGenCost, singleGenCost int,
) *GenerationHandler {
	return &GenerationHandler{
		projectRepo:     projectRepo,
		monthRepo:       monthRepo,
		jobRepo:         jobRepo,
		tokenRepo:       tokenRepo,
		genWorker:       genWorker,
		webhookAdapters: webhookAdapters,
		webhookSecrets:  webhookSecrets,
		fullGenCost:     fullGenCost,
		singleGenCost:   singleGenCost,
	}
}

type monthInput struct {
	Month             int    `json:"month"`
	ReferenceImageURL string `json:"reference_image_url"`
	AssetID           string `json:"asset_id,omitempty"`
}

type generateRequest struct {
	Months []monthInput `json:"months"`
}

type generateResponse struct {
	JobIDs           []string `json:"job_ids"`
	EstimatedSeconds int      `json:"estimated_seconds"`
}

// POST /api/v1/dev/generate
// Temporary unauthenticated endpoint that returns a canned generateResponse.
// Lets the iOS app verify the network round-trip before the real imagegen
// pipeline is wired up. Remove once production generate flow is exercised
// end-to-end from the client.
func (h *GenerationHandler) DevGenerateStub(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	count := len(req.Months)
	if count == 0 {
		count = 12
	}
	jobIDs := make([]string, count)
	for i := 0; i < count; i++ {
		jobIDs[i] = fmt.Sprintf("dev-job-%d", i+1)
	}

	slog.Info("dev generate stub hit", "month_count", count)
	writeJSON(w, http.StatusAccepted, generateResponse{
		JobIDs:           jobIDs,
		EstimatedSeconds: 30,
	})
}

// POST /api/v1/projects/:id/generate
func (h *GenerationHandler) GenerateCalendar(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := chi.URLParam(r, "id")
	project, err := h.projectRepo.FindByID(r.Context(), projectID)
	if err != nil || project == nil || project.UserID != userID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Months) == 0 {
		writeError(w, http.StatusBadRequest, "months array is required")
		return
	}

	// Atomic token deduction
	if err := h.tokenRepo.DeductAtomic(r.Context(), userID, h.fullGenCost,
		fmt.Sprintf("full calendar generation for project %s", projectID)); err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error())
		return
	}

	// Upsert months + create generation jobs
	var jobIDs []string
	for _, mi := range req.Months {
		month := &domain.CalendarMonth{
			ProjectID:             projectID,
			Month:                 mi.Month,
			ReferencePhotoAssetID: mi.AssetID,
			ReferenceImageURL:     mi.ReferenceImageURL,
			Status:                domain.MonthStatusGenerating,
		}
		if err := h.monthRepo.Upsert(r.Context(), month); err != nil {
			slog.Error("upsert month failed", "month", mi.Month, "error", err)
			continue
		}

		job := &domain.GenerationJob{
			UserID:     userID,
			CalendarID: projectID,
			MonthID:    month.ID,
			Status:     domain.JobStatusQueued,
		}
		if err := h.jobRepo.Create(r.Context(), job); err != nil {
			slog.Error("create job failed", "month", mi.Month, "error", err)
			continue
		}

		jobIDs = append(jobIDs, job.ID)
		h.genWorker.EnqueueJob(job.ID)
	}

	// Update project status
	project.Status = domain.ProjectStatusGenerating
	_ = h.projectRepo.Update(r.Context(), project)

	writeJSON(w, http.StatusAccepted, generateResponse{
		JobIDs:           jobIDs,
		EstimatedSeconds: 90,
	})
}

// POST /api/v1/projects/:id/months/:month/regenerate
func (h *GenerationHandler) RegenerateMonth(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := chi.URLParam(r, "id")
	project, err := h.projectRepo.FindByID(r.Context(), projectID)
	if err != nil || project == nil || project.UserID != userID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		ReferenceImageURL string `json:"reference_image_url"`
		Prompt            string `json:"prompt"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.tokenRepo.DeductAtomic(r.Context(), userID, h.singleGenCost,
		fmt.Sprintf("regenerate month in project %s", projectID)); err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error())
		return
	}

	// Find the existing month
	months, err := h.monthRepo.FindByProjectID(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch months")
		return
	}

	monthNum := 0
	fmt.Sscanf(chi.URLParam(r, "month"), "%d", &monthNum)

	var targetMonth *domain.CalendarMonth
	for _, m := range months {
		if m.Month == monthNum {
			targetMonth = m
			break
		}
	}
	if targetMonth == nil {
		writeError(w, http.StatusNotFound, "month not found")
		return
	}

	// Apply any overrides
	if body.ReferenceImageURL != "" || body.Prompt != "" {
		refURL := targetMonth.ReferenceImageURL
		if body.ReferenceImageURL != "" {
			refURL = body.ReferenceImageURL
		}
		_ = h.monthRepo.UpdatePromptAndRef(r.Context(), targetMonth.ID, body.Prompt, refURL)
		targetMonth.Prompt = body.Prompt
		targetMonth.ReferenceImageURL = refURL
	}

	_ = h.monthRepo.UpdateGeneratedImage(r.Context(), targetMonth.ID, "", domain.MonthStatusGenerating)

	job := &domain.GenerationJob{
		UserID:     userID,
		CalendarID: projectID,
		MonthID:    targetMonth.ID,
		Status:     domain.JobStatusQueued,
	}
	if err := h.jobRepo.Create(r.Context(), job); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	h.genWorker.EnqueueJob(job.ID)

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": job.ID})
}

// GET /api/v1/jobs/:id
func (h *GenerationHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jobID := chi.URLParam(r, "id")
	job, err := h.jobRepo.FindByID(r.Context(), jobID)
	if err != nil || job == nil || job.UserID != userID {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":               job.ID,
		"status":           string(job.Status),
		"result_image_url": job.ResultImageURL,
		"error":            job.ErrorMessage,
		"month_id":         job.MonthID,
		"month":            job.Month,
		"calendar_id":      job.CalendarID,
	})
}

// POST /api/v1/webhooks/imagegen/{provider}
// Called by an image generation provider when a job changes state.
// Provider-specific parsing + signature verification lives in WebhookAdapter
// implementations; this handler stays provider-agnostic.
func (h *GenerationHandler) ImageGenWebhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	adapter, ok := h.webhookAdapters[provider]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown provider")
		return
	}

	secret := h.webhookSecrets[provider]
	if err := adapter.VerifySignature(r, secret); err != nil {
		slog.Warn("imagegen webhook signature verification failed", "provider", provider, "error", err)
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	evt, err := adapter.ParseEvent(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	job, err := h.jobRepo.FindByProviderJobID(r.Context(), provider, evt.ProviderJobID)
	if err != nil || job == nil {
		// Unknown job — could be from a different environment. Acknowledge silently.
		w.WriteHeader(http.StatusOK)
		return
	}

	switch evt.Status {
	case domain.JobStatusComplete:
		// Worker's poll loop handles download + S3 storage. The webhook here is
		// informational; promoting it to the fast path is a separate change.
		slog.Info("imagegen webhook: job completed", "provider", provider, "provider_job_id", evt.ProviderJobID, "job_id", job.ID)
	case domain.JobStatusFailed:
		_ = h.jobRepo.UpdateStatus(r.Context(), job.ID, domain.JobStatusFailed, "", evt.ErrorMessage)
		_ = h.monthRepo.UpdateGeneratedImage(r.Context(), job.MonthID, "", domain.MonthStatusFailed)
	}

	w.WriteHeader(http.StatusOK)
}
