package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	v1 "github.com/brycesharrits/fam-cal-insta/internal/api/v1"
	"github.com/brycesharrits/fam-cal-insta/internal/auth"
)

type Router struct {
	authHandler       *v1.AuthHandler
	projectHandler    *v1.ProjectHandler
	generationHandler *v1.GenerationHandler
	uploadHandler     *v1.UploadHandler
	tokenHandler      *v1.TokenHandler
	orderHandler      *v1.OrderHandler
	jwtSvc            *auth.JWTService
}

func NewRouter(
	authHandler *v1.AuthHandler,
	projectHandler *v1.ProjectHandler,
	generationHandler *v1.GenerationHandler,
	uploadHandler *v1.UploadHandler,
	tokenHandler *v1.TokenHandler,
	orderHandler *v1.OrderHandler,
	jwtSvc *auth.JWTService,
) *Router {
	return &Router{
		authHandler:       authHandler,
		projectHandler:    projectHandler,
		generationHandler: generationHandler,
		uploadHandler:     uploadHandler,
		tokenHandler:      tokenHandler,
		orderHandler:      orderHandler,
		jwtSvc:            jwtSvc,
	}
}

func (ro *Router) Build() http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(apimiddleware.CORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public — no auth required
		r.Post("/auth/apple", ro.authHandler.AppleSignIn)
		r.Post("/webhooks/imagegen/{provider}", ro.generationHandler.ImageGenWebhook)
		r.Post("/webhooks/print-partner", ro.orderHandler.PrintPartnerWebhook)
		r.Get("/tokens/products", ro.tokenHandler.GetProducts)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(apimiddleware.Authenticate(ro.jwtSvc))

			// Users
			r.Get("/users/me", ro.authHandler.GetMe)

			// Projects
			r.Post("/projects", ro.projectHandler.Create)
			r.Get("/projects", ro.projectHandler.List)
			r.Get("/projects/{id}", ro.projectHandler.Get)
			r.Patch("/projects/{id}", ro.projectHandler.Update)
			r.Delete("/projects/{id}", ro.projectHandler.Delete)

			// Generation
			r.Post("/projects/{id}/generate", ro.generationHandler.GenerateCalendar)
			r.Post("/projects/{id}/months/{month}/regenerate", ro.generationHandler.RegenerateMonth)
			r.Get("/jobs/{id}", ro.generationHandler.GetJob)

			// Uploads
			r.Post("/uploads/presign", ro.uploadHandler.Presign)

			// Tokens / IAP
			r.Get("/tokens/balance", ro.tokenHandler.GetBalance)
			r.Get("/tokens/history", ro.tokenHandler.GetHistory)
			r.Post("/tokens/verify-purchase", ro.tokenHandler.VerifyPurchase)

			// Orders
			r.Post("/projects/{id}/orders/print", ro.orderHandler.SubmitPrintOrder)
			r.Post("/projects/{id}/orders/pdf-export", ro.orderHandler.ExportPDF)
			r.Get("/orders/{id}", ro.orderHandler.GetOrder)
		})
	})

	return r
}
