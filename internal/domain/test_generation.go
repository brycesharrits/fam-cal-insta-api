package domain

import "time"

type TestGenerationMode string

const (
	TestGenerationModeText TestGenerationMode = "text"
	TestGenerationModeEdit TestGenerationMode = "edit"
)

type TestGenerationStatus string

const (
	TestGenerationStatusPending  TestGenerationStatus = "pending"
	TestGenerationStatusComplete TestGenerationStatus = "complete"
	TestGenerationStatusFailed   TestGenerationStatus = "failed"
)

type TestGeneration struct {
	ID               string               `db:"id"`
	UserID           string               `db:"user_id"`
	Mode             TestGenerationMode   `db:"mode"`
	Prompt           string               `db:"prompt"`
	InputImageBytes  []byte               `db:"input_image_bytes"`
	InputImageMime   string               `db:"input_image_mime"`
	OutputImageBytes []byte               `db:"output_image_bytes"`
	OutputImageMime  string               `db:"output_image_mime"`
	Status           TestGenerationStatus `db:"status"`
	ErrorMessage     string               `db:"error_message"`
	DurationMs       int                  `db:"duration_ms"`
	CreatedAt        time.Time            `db:"created_at"`
}
