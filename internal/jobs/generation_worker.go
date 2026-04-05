package jobs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/imagegen"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
	"github.com/brycesharrits/fam-cal-insta/internal/service"
	"github.com/brycesharrits/fam-cal-insta/internal/storage"
)

// GenerationWorker handles the async image generation lifecycle.
type GenerationWorker struct {
	provider      imagegen.ImageGenerationProvider
	storage       storage.ObjectStorage
	jobRepo       repository.GenerationJobRepository
	monthRepo     repository.MonthRepository
	promptBuilder *service.PromptBuilder
	worker        *Worker
	httpClient    *http.Client
}

func NewGenerationWorker(
	provider imagegen.ImageGenerationProvider,
	storage storage.ObjectStorage,
	jobRepo repository.GenerationJobRepository,
	monthRepo repository.MonthRepository,
	promptBuilder *service.PromptBuilder,
	concurrency int,
) *GenerationWorker {
	return &GenerationWorker{
		provider:      provider,
		storage:       storage,
		jobRepo:       jobRepo,
		monthRepo:     monthRepo,
		promptBuilder: promptBuilder,
		worker:        NewWorker(concurrency),
		httpClient:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *GenerationWorker) Start(ctx context.Context) {
	g.worker.Start(ctx)
}

func (g *GenerationWorker) Stop() {
	g.worker.Stop()
}

// EnqueueJob adds a generation job to the worker queue.
func (g *GenerationWorker) EnqueueJob(jobID string) {
	g.worker.Enqueue(func(ctx context.Context) {
		if err := g.processJob(ctx, jobID); err != nil {
			slog.Error("generation job failed", "job_id", jobID, "error", err)
		}
	})
}

func (g *GenerationWorker) processJob(ctx context.Context, jobID string) error {
	job, err := g.jobRepo.FindByID(ctx, jobID)
	if err != nil || job == nil {
		return fmt.Errorf("finding job %s: %w", jobID, err)
	}

	month, err := g.monthRepo.FindByID(ctx, job.MonthID)
	if err != nil || month == nil {
		return fmt.Errorf("finding month %s: %w", job.MonthID, err)
	}

	// Build prompt — referenceContext derived from season/month for now.
	// In the future, this could include AI analysis of the reference image.
	referenceContext := buildReferenceContext(month)
	prompt := g.promptBuilder.Build("watercolor", month.Month, 2026, referenceContext)
	if month.Prompt != "" {
		prompt = month.Prompt // user-overridden prompt takes precedence
	}

	// Submit to Replicate
	predID, err := g.provider.GenerateAsync(ctx, imagegen.GenerationRequest{
		ReferenceImageURL: month.ReferenceImageURL,
		Prompt:            prompt,
		Width:             2400,
		Height:            1800,
		JobID:             jobID,
	})
	if err != nil {
		_ = g.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusFailed, "", err.Error())
		_ = g.monthRepo.UpdateGeneratedImage(ctx, job.MonthID, "", domain.MonthStatusFailed)
		return fmt.Errorf("submitting to replicate: %w", err)
	}

	if err := g.jobRepo.UpdatePredictionID(ctx, jobID, predID); err != nil {
		return fmt.Errorf("storing prediction id: %w", err)
	}

	// Poll for result (webhook is the preferred path, polling is the fallback)
	result, err := g.pollUntilDone(ctx, predID)
	if err != nil {
		_ = g.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusFailed, "", err.Error())
		_ = g.monthRepo.UpdateGeneratedImage(ctx, job.MonthID, "", domain.MonthStatusFailed)
		return err
	}

	// Download image from Replicate CDN and store in our S3
	// Critical: Replicate URLs expire — we must persist to our own storage.
	s3URL, err := g.downloadAndStore(ctx, result.ImageURL, job.CalendarID, month.Month)
	if err != nil {
		return fmt.Errorf("storing generated image: %w", err)
	}

	if err := g.jobRepo.UpdateStatus(ctx, jobID, domain.JobStatusComplete, s3URL, ""); err != nil {
		return err
	}
	return g.monthRepo.UpdateGeneratedImage(ctx, job.MonthID, s3URL, domain.MonthStatusComplete)
}

func (g *GenerationWorker) pollUntilDone(ctx context.Context, predID string) (*imagegen.GenerationResult, error) {
	backoff := 2 * time.Second
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		result, done, err := g.provider.PollStatus(ctx, predID)
		if err != nil {
			return nil, err
		}
		if done {
			return result, nil
		}

		if backoff < 10*time.Second {
			backoff += time.Second
		}
	}
}

func (g *GenerationWorker) downloadAndStore(ctx context.Context, imageURL, calendarID string, month int) (string, error) {
	resp, err := g.httpClient.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading image: %w", err)
	}

	key := fmt.Sprintf("calendars/%s/months/%02d_%d.jpg", calendarID, month, time.Now().Unix())
	url, err := g.storage.PutObject(ctx, key, strings.NewReader(string(data)), "image/jpeg")
	if err != nil {
		return "", fmt.Errorf("uploading to storage: %w", err)
	}

	return url, nil
}

func buildReferenceContext(month *domain.CalendarMonth) string {
	if month.Prompt != "" {
		return month.Prompt
	}
	return fmt.Sprintf("a family memory from month %d", month.Month)
}
