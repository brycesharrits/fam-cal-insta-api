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
	Month                 int       `db:"month"` // 1-12, joined from calendar_months
	Status                JobStatus `db:"status"`
	Provider              string    `db:"provider"`
	ProviderJobID         string    `db:"provider_job_id"`
	ResultImageURL        string    `db:"result_image_url"`
	ErrorMessage          string    `db:"error_message"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}
