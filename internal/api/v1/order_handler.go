package v1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/printpartner"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

type OrderHandler struct {
	projectRepo  repository.ProjectRepository
	monthRepo    repository.MonthRepository
	orderRepo    repository.OrderRepository
	tokenRepo    repository.TokenRepository
	printPartner printpartner.PrintPartner
	pdfExportCost int
}

func NewOrderHandler(
	projectRepo repository.ProjectRepository,
	monthRepo repository.MonthRepository,
	orderRepo repository.OrderRepository,
	tokenRepo repository.TokenRepository,
	partner printpartner.PrintPartner,
	pdfExportCost int,
) *OrderHandler {
	return &OrderHandler{
		projectRepo:   projectRepo,
		monthRepo:     monthRepo,
		orderRepo:     orderRepo,
		tokenRepo:     tokenRepo,
		printPartner:  partner,
		pdfExportCost: pdfExportCost,
	}
}

type addressInput struct {
	Name       string `json:"name"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// POST /api/v1/projects/:id/orders/print
func (h *OrderHandler) SubmitPrintOrder(w http.ResponseWriter, r *http.Request) {
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
		ShippingAddress addressInput `json:"shipping_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	months, err := h.monthRepo.FindByProjectID(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch months")
		return
	}

	var monthImages []printpartner.MonthImage
	for _, m := range months {
		if m.GeneratedImageURL != "" {
			monthImages = append(monthImages, printpartner.MonthImage{
				Month:    m.Month,
				ImageURL: m.GeneratedImageURL,
			})
		}
	}

	if len(monthImages) < 12 {
		writeError(w, http.StatusBadRequest, "calendar is not complete — not all months have generated images")
		return
	}

	// Create order record
	order := &domain.Order{
		UserID:     userID,
		CalendarID: projectID,
		Partner:    h.printPartner.PartnerName(),
		Status:     domain.OrderStatusPending,
	}
	if err := h.orderRepo.Create(r.Context(), order); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	// Submit to print partner
	result, err := h.printPartner.SubmitOrder(r.Context(), printpartner.CalendarPayload{
		ProjectID:   projectID,
		Year:        project.Year,
		MonthImages: monthImages,
		ShippingAddr: printpartner.Address{
			Name:       body.ShippingAddress.Name,
			Line1:      body.ShippingAddress.Line1,
			Line2:      body.ShippingAddress.Line2,
			City:       body.ShippingAddress.City,
			State:      body.ShippingAddress.State,
			PostalCode: body.ShippingAddress.PostalCode,
			Country:    body.ShippingAddress.Country,
		},
	})
	if err != nil {
		_ = h.orderRepo.UpdateStatus(r.Context(), order.ID, string(domain.OrderStatusFailed), "", "")
		writeError(w, http.StatusInternalServerError, "failed to submit order to print partner")
		return
	}

	_ = h.orderRepo.UpdateStatus(r.Context(), order.ID, string(domain.OrderStatusProcessing),
		result.PartnerOrderID, result.TrackingURL)

	// Update project status
	project.Status = domain.ProjectStatusOrdered
	_ = h.projectRepo.Update(r.Context(), project)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"order_id":         order.ID,
		"partner_order_id": result.PartnerOrderID,
		"status":           "processing",
		"est_delivery":     result.EstDelivery,
	})
}

// POST /api/v1/projects/:id/orders/pdf-export
func (h *OrderHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
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

	if err := h.tokenRepo.DeductAtomic(r.Context(), userID, h.pdfExportCost,
		fmt.Sprintf("PDF export for project %s", projectID)); err != nil {
		writeError(w, http.StatusPaymentRequired, err.Error())
		return
	}

	// TODO: Implement actual PDF generation (collect month image URLs, render PDF)
	// For now, return a placeholder response
	writeJSON(w, http.StatusOK, map[string]string{
		"download_url": "https://placeholder.example.com/pdf/" + projectID,
		"expires_at":   "2026-12-31T00:00:00Z",
		"message":      "PDF generation coming soon",
	})
}

// GET /api/v1/orders/:id
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orderID := chi.URLParam(r, "id")
	order, err := h.orderRepo.FindByID(r.Context(), orderID)
	if err != nil || order == nil || order.UserID != userID {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":               order.ID,
		"calendar_id":      order.CalendarID,
		"partner":          order.Partner,
		"status":           order.Status,
		"partner_order_id": order.PartnerOrderID,
		"tracking_url":     order.TrackingURL,
		"created_at":       order.CreatedAt,
	})
}

// POST /api/v1/webhooks/print-partner
func (h *OrderHandler) PrintPartnerWebhook(w http.ResponseWriter, r *http.Request) {
	// Each partner has a different webhook format.
	// Parse the mock format for now.
	var payload struct {
		PartnerOrderID string `json:"partner_order_id"`
		Status         string `json:"status"`
		TrackingURL    string `json:"tracking_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusOK) // always 200 to acknowledge
		return
	}

	// TODO: Look up order by partner_order_id and update status
	w.WriteHeader(http.StatusOK)
}
