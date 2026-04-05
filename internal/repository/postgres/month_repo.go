package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type MonthRepo struct {
	db *pgxpool.Pool
}

func NewMonthRepo(db *pgxpool.Pool) *MonthRepo {
	return &MonthRepo{db: db}
}

func (r *MonthRepo) Upsert(ctx context.Context, m *domain.CalendarMonth) error {
	query := `
		INSERT INTO calendar_months (project_id, month, reference_photo_asset_id, reference_image_url, prompt, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, month) DO UPDATE SET
			reference_photo_asset_id = EXCLUDED.reference_photo_asset_id,
			reference_image_url      = EXCLUDED.reference_image_url,
			prompt                   = EXCLUDED.prompt,
			status                   = EXCLUDED.status,
			updated_at               = NOW()
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		m.ProjectID, m.Month, m.ReferencePhotoAssetID, m.ReferenceImageURL, m.Prompt, m.Status,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *MonthRepo) FindByProjectID(ctx context.Context, projectID string) ([]*domain.CalendarMonth, error) {
	query := `
		SELECT id, project_id, month, reference_photo_asset_id, reference_image_url, prompt, generated_image_url, status, created_at, updated_at
		FROM calendar_months WHERE project_id = $1 ORDER BY month ASC`
	rows, err := r.db.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []*domain.CalendarMonth
	for rows.Next() {
		m := &domain.CalendarMonth{}
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Month, &m.ReferencePhotoAssetID,
			&m.ReferenceImageURL, &m.Prompt, &m.GeneratedImageURL, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		months = append(months, m)
	}
	return months, rows.Err()
}

func (r *MonthRepo) FindByID(ctx context.Context, id string) (*domain.CalendarMonth, error) {
	query := `
		SELECT id, project_id, month, reference_photo_asset_id, reference_image_url, prompt, generated_image_url, status, created_at, updated_at
		FROM calendar_months WHERE id = $1`
	m := &domain.CalendarMonth{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.ProjectID, &m.Month, &m.ReferencePhotoAssetID,
		&m.ReferenceImageURL, &m.Prompt, &m.GeneratedImageURL, &m.Status, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

func (r *MonthRepo) UpdateGeneratedImage(ctx context.Context, id, imageURL string, status domain.MonthStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE calendar_months SET generated_image_url=$1, status=$2, updated_at=NOW() WHERE id=$3`,
		imageURL, status, id,
	)
	return err
}

func (r *MonthRepo) UpdatePromptAndRef(ctx context.Context, id, prompt, refImageURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE calendar_months SET prompt=$1, reference_image_url=$2, updated_at=NOW() WHERE id=$3`,
		prompt, refImageURL, id,
	)
	return err
}
