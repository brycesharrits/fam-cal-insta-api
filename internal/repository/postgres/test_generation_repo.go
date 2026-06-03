package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type TestGenerationRepo struct {
	db *pgxpool.Pool
}

func NewTestGenerationRepo(db *pgxpool.Pool) *TestGenerationRepo {
	return &TestGenerationRepo{db: db}
}

func (r *TestGenerationRepo) Create(ctx context.Context, t *domain.TestGeneration) error {
	query := `
		INSERT INTO test_generations
			(user_id, mode, prompt,
			 input_image_bytes, input_image_mime,
			 output_image_bytes, output_image_mime,
			 status, error_message, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	return r.db.QueryRow(ctx, query,
		t.UserID, t.Mode, t.Prompt,
		nullableBytes(t.InputImageBytes), nullableString(t.InputImageMime),
		nullableBytes(t.OutputImageBytes), nullableString(t.OutputImageMime),
		t.Status, nullableString(t.ErrorMessage), nullableInt(t.DurationMs),
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *TestGenerationRepo) FindByID(ctx context.Context, id string) (*domain.TestGeneration, error) {
	query := `
		SELECT id, user_id, mode, prompt,
		       input_image_bytes, input_image_mime,
		       output_image_bytes, output_image_mime,
		       status, COALESCE(error_message, ''), COALESCE(duration_ms, 0), created_at
		  FROM test_generations
		 WHERE id = $1`
	t := &domain.TestGeneration{}
	var inputMime, outputMime *string
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.Mode, &t.Prompt,
		&t.InputImageBytes, &inputMime,
		&t.OutputImageBytes, &outputMime,
		&t.Status, &t.ErrorMessage, &t.DurationMs, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if inputMime != nil {
		t.InputImageMime = *inputMime
	}
	if outputMime != nil {
		t.OutputImageMime = *outputMime
	}
	return t, nil
}

func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(n int) any {
	if n == 0 {
		return nil
	}
	return n
}
