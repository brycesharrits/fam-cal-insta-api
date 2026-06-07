package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type GenerationJobRepo struct {
	db *pgxpool.Pool
}

func NewGenerationJobRepo(db *pgxpool.Pool) *GenerationJobRepo {
	return &GenerationJobRepo{db: db}
}

const generationJobSelect = `
	SELECT gj.id, gj.user_id, gj.calendar_id, gj.month_id, cm.month,
	       gj.status, gj.provider, gj.provider_job_id,
	       gj.result_image_url, gj.error_message, gj.created_at, gj.updated_at
	FROM generation_jobs gj
	JOIN calendar_months cm ON cm.id = gj.month_id`

func scanGenerationJob(row pgx.Row, job *domain.GenerationJob) error {
	var provider, providerJobID *string
	if err := row.Scan(
		&job.ID, &job.UserID, &job.CalendarID, &job.MonthID, &job.Month,
		&job.Status, &provider, &providerJobID,
		&job.ResultImageURL, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	); err != nil {
		return err
	}
	if provider != nil {
		job.Provider = *provider
	}
	if providerJobID != nil {
		job.ProviderJobID = *providerJobID
	}
	return nil
}

func (r *GenerationJobRepo) Create(ctx context.Context, job *domain.GenerationJob) error {
	query := `
		INSERT INTO generation_jobs (user_id, calendar_id, month_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query, job.UserID, job.CalendarID, job.MonthID, job.Status).
		Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *GenerationJobRepo) FindByID(ctx context.Context, id string) (*domain.GenerationJob, error) {
	job := &domain.GenerationJob{}
	err := scanGenerationJob(r.db.QueryRow(ctx, generationJobSelect+` WHERE gj.id = $1`, id), job)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *GenerationJobRepo) FindByProviderJobID(ctx context.Context, provider, providerJobID string) (*domain.GenerationJob, error) {
	job := &domain.GenerationJob{}
	err := scanGenerationJob(r.db.QueryRow(ctx,
		generationJobSelect+` WHERE gj.provider = $1 AND gj.provider_job_id = $2`,
		provider, providerJobID,
	), job)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *GenerationJobRepo) FindByCalendarID(ctx context.Context, calendarID string) ([]*domain.GenerationJob, error) {
	rows, err := r.db.Query(ctx,
		generationJobSelect+` WHERE gj.calendar_id = $1 ORDER BY gj.created_at ASC`,
		calendarID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.GenerationJob
	for rows.Next() {
		job := &domain.GenerationJob{}
		if err := scanGenerationJob(rows, job); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *GenerationJobRepo) UpdateStatus(ctx context.Context, id string, status domain.JobStatus, resultURL, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE generation_jobs SET status=$1, result_image_url=$2, error_message=$3, updated_at=NOW() WHERE id=$4`,
		status, resultURL, errMsg, id,
	)
	return err
}

func (r *GenerationJobRepo) UpdateProviderJobID(ctx context.Context, id, provider, providerJobID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE generation_jobs SET provider=$1, provider_job_id=$2, status='processing', updated_at=NOW() WHERE id=$3`,
		provider, providerJobID, id,
	)
	return err
}
