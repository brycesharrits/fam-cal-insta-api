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

func (r *GenerationJobRepo) Create(ctx context.Context, job *domain.GenerationJob) error {
	query := `
		INSERT INTO generation_jobs (user_id, calendar_id, month_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query, job.UserID, job.CalendarID, job.MonthID, job.Status).
		Scan(&job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *GenerationJobRepo) FindByID(ctx context.Context, id string) (*domain.GenerationJob, error) {
	query := `SELECT id, user_id, calendar_id, month_id, status, replicate_prediction_id, result_image_url, error_message, created_at, updated_at FROM generation_jobs WHERE id = $1`
	job := &domain.GenerationJob{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.UserID, &job.CalendarID, &job.MonthID, &job.Status,
		&job.ReplicatePredictionID, &job.ResultImageURL, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *GenerationJobRepo) FindByReplicatePredictionID(ctx context.Context, predictionID string) (*domain.GenerationJob, error) {
	query := `SELECT id, user_id, calendar_id, month_id, status, replicate_prediction_id, result_image_url, error_message, created_at, updated_at FROM generation_jobs WHERE replicate_prediction_id = $1`
	job := &domain.GenerationJob{}
	err := r.db.QueryRow(ctx, query, predictionID).Scan(
		&job.ID, &job.UserID, &job.CalendarID, &job.MonthID, &job.Status,
		&job.ReplicatePredictionID, &job.ResultImageURL, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *GenerationJobRepo) FindByCalendarID(ctx context.Context, calendarID string) ([]*domain.GenerationJob, error) {
	query := `SELECT id, user_id, calendar_id, month_id, status, replicate_prediction_id, result_image_url, error_message, created_at, updated_at FROM generation_jobs WHERE calendar_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, query, calendarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.GenerationJob
	for rows.Next() {
		job := &domain.GenerationJob{}
		if err := rows.Scan(&job.ID, &job.UserID, &job.CalendarID, &job.MonthID, &job.Status,
			&job.ReplicatePredictionID, &job.ResultImageURL, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt); err != nil {
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

func (r *GenerationJobRepo) UpdatePredictionID(ctx context.Context, id, predictionID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE generation_jobs SET replicate_prediction_id=$1, status='processing', updated_at=NOW() WHERE id=$2`,
		predictionID, id,
	)
	return err
}
