package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type ProjectRepo struct {
	db *pgxpool.Pool
}

func NewProjectRepo(db *pgxpool.Pool) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(ctx context.Context, p *domain.CalendarProject) error {
	query := `
		INSERT INTO calendar_projects (user_id, name, year, theme, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query, p.UserID, p.Name, p.Year, p.Theme, p.Status).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ProjectRepo) FindByID(ctx context.Context, id string) (*domain.CalendarProject, error) {
	query := `SELECT id, user_id, name, year, theme, status, created_at, updated_at FROM calendar_projects WHERE id = $1`
	p := &domain.CalendarProject{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Year, &p.Theme, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *ProjectRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.CalendarProject, error) {
	query := `SELECT id, user_id, name, year, theme, status, created_at, updated_at FROM calendar_projects WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*domain.CalendarProject
	for rows.Next() {
		p := &domain.CalendarProject{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Year, &p.Theme, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) Update(ctx context.Context, p *domain.CalendarProject) error {
	query := `UPDATE calendar_projects SET name=$1, theme=$2, status=$3, updated_at=NOW() WHERE id=$4 AND user_id=$5`
	_, err := r.db.Exec(ctx, query, p.Name, p.Theme, p.Status, p.ID, p.UserID)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM calendar_projects WHERE id = $1`, id)
	return err
}
