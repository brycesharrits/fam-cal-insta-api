package domain

import "time"

type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusProcessing JobStatus = "processing"
	JobStatusComplete   JobStatus = "complete"
	JobStatusFailed     JobStatus = "failed"
)

type GenerationJob struct {
	ID                    string    `db:"id"`
	UserID                string    `db:"user_id"`
	CalendarID            string    `db:"calendar_id"`
	MonthID               string    `db:"month_id"`
	Status                JobStatus `db:"status"`
	ReplicatePredictionID string    `db:"replicate_prediction_id"`
	ResultImageURL        string    `db:"result_image_url"`
	ErrorMessage          string    `db:"error_message"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}
