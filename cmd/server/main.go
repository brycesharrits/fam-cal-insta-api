package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/api"
	v1 "github.com/brycesharrits/fam-cal-insta/internal/api/v1"
	"github.com/brycesharrits/fam-cal-insta/internal/auth"
	"github.com/brycesharrits/fam-cal-insta/internal/config"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen/replicate"
	"github.com/brycesharrits/fam-cal-insta/internal/jobs"
	"github.com/brycesharrits/fam-cal-insta/internal/printpartner/mock"
	"github.com/brycesharrits/fam-cal-insta/internal/repository/postgres"
	"github.com/brycesharrits/fam-cal-insta/internal/service"
	"github.com/brycesharrits/fam-cal-insta/internal/storage/s3"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database
	db, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Repositories
	userRepo := postgres.NewUserRepo(db)
	projectRepo := postgres.NewProjectRepo(db)
	monthRepo := postgres.NewMonthRepo(db)
	jobRepo := postgres.NewGenerationJobRepo(db)
	tokenRepo := postgres.NewTokenRepo(db)
	orderRepo := postgres.NewOrderRepo(db)

	// Auth
	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiry)
	appleVerifier := auth.NewAppleSignInVerifier()

	// Storage
	objectStorage, err := s3.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	// Image generation
	fluxProvider := replicate.NewFluxProvider(cfg.ReplicateAPIKey, cfg.ReplicateFluxModel)
	replicateWebhookAdapter := replicate.NewWebhookAdapter()
	webhookAdapters := map[string]imagegen.WebhookAdapter{
		replicateWebhookAdapter.Name(): replicateWebhookAdapter,
	}
	webhookSecrets := map[string]string{
		replicateWebhookAdapter.Name(): cfg.ReplicateWebhookSecret,
	}
	promptBuilder := service.NewPromptBuilder()

	// Print partner (mock until real partner is selected)
	printPartner := mock.New()

	// Generation worker
	genWorker := jobs.NewGenerationWorker(fluxProvider, objectStorage, jobRepo, monthRepo, promptBuilder, 5)
	genWorker.Start(ctx)
	defer genWorker.Stop()

	// Handlers
	authHandler := v1.NewAuthHandler(appleVerifier, jwtSvc, userRepo)
	projectHandler := v1.NewProjectHandler(projectRepo, monthRepo)
	generationHandler := v1.NewGenerationHandler(
		projectRepo, monthRepo, jobRepo, tokenRepo, genWorker,
		webhookAdapters, webhookSecrets,
		cfg.TokenCosts.FullCalendarGeneration,
		cfg.TokenCosts.SingleMonthRegeneration,
	)
	uploadHandler := v1.NewUploadHandler(objectStorage)
	tokenHandler := v1.NewTokenHandler(userRepo, tokenRepo)
	orderHandler := v1.NewOrderHandler(
		projectRepo, monthRepo, orderRepo, tokenRepo, printPartner,
		cfg.TokenCosts.PDFExport,
	)

	// Router
	router := api.NewRouter(authHandler, projectHandler, generationHandler, uploadHandler, tokenHandler, orderHandler, jwtSvc)
	handler := router.Build()

	// HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server stopped")
}
