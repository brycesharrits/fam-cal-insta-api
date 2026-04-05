package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

type ProjectHandler struct {
	projectRepo repository.ProjectRepository
	monthRepo   repository.MonthRepository
}

func NewProjectHandler(projectRepo repository.ProjectRepository, monthRepo repository.MonthRepository) *ProjectHandler {
	return &ProjectHandler{projectRepo: projectRepo, monthRepo: monthRepo}
}

type projectDTO struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Year      int        `json:"year"`
	Theme     string     `json:"theme"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Months    []monthDTO `json:"months,omitempty"`
}

type monthDTO struct {
	ID                    string `json:"id"`
	Month                 int    `json:"month"`
	ReferencePhotoAssetID string `json:"reference_photo_asset_id,omitempty"`
	ReferenceImageURL     string `json:"reference_image_url,omitempty"`
	Prompt                string `json:"prompt,omitempty"`
	GeneratedImageURL     string `json:"generated_image_url,omitempty"`
	Status                string `json:"status"`
}

// POST /api/v1/projects
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		Name  string `json:"name"`
		Year  int    `json:"year"`
		Theme string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" || body.Year == 0 || body.Theme == "" {
		writeError(w, http.StatusBadRequest, "name, year, and theme are required")
		return
	}

	p := &domain.CalendarProject{
		UserID: userID,
		Name:   body.Name,
		Year:   body.Year,
		Theme:  body.Theme,
		Status: domain.ProjectStatusDraft,
	}
	if err := h.projectRepo.Create(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	writeJSON(w, http.StatusCreated, toProjectDTO(p, nil))
}

// GET /api/v1/projects
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projects, err := h.projectRepo.FindByUserID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch projects")
		return
	}

	dtos := make([]projectDTO, 0, len(projects))
	for _, p := range projects {
		dtos = append(dtos, toProjectDTO(p, nil))
	}
	writeJSON(w, http.StatusOK, dtos)
}

// GET /api/v1/projects/:id
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	p, err := h.projectRepo.FindByID(r.Context(), id)
	if err != nil || p == nil || p.UserID != userID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	months, err := h.monthRepo.FindByProjectID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch months")
		return
	}

	writeJSON(w, http.StatusOK, toProjectDTO(p, months))
}

// PATCH /api/v1/projects/:id
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	p, err := h.projectRepo.FindByID(r.Context(), id)
	if err != nil || p == nil || p.UserID != userID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Name  *string `json:"name"`
		Theme *string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name != nil {
		p.Name = *body.Name
	}
	if body.Theme != nil {
		p.Theme = *body.Theme
	}

	if err := h.projectRepo.Update(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}

	writeJSON(w, http.StatusOK, toProjectDTO(p, nil))
}

// DELETE /api/v1/projects/:id
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	p, err := h.projectRepo.FindByID(r.Context(), id)
	if err != nil || p == nil || p.UserID != userID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	if err := h.projectRepo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toProjectDTO(p *domain.CalendarProject, months []*domain.CalendarMonth) projectDTO {
	dto := projectDTO{
		ID:        p.ID,
		Name:      p.Name,
		Year:      p.Year,
		Theme:     p.Theme,
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	for _, m := range months {
		dto.Months = append(dto.Months, monthDTO{
			ID:                    m.ID,
			Month:                 m.Month,
			ReferencePhotoAssetID: m.ReferencePhotoAssetID,
			ReferenceImageURL:     m.ReferenceImageURL,
			Prompt:                m.Prompt,
			GeneratedImageURL:     m.GeneratedImageURL,
			Status:                string(m.Status),
		})
	}
	return dto
}
