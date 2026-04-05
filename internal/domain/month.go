package domain

import "time"

type MonthStatus string

const (
	MonthStatusPending    MonthStatus = "pending"
	MonthStatusGenerating MonthStatus = "generating"
	MonthStatusComplete   MonthStatus = "complete"
	MonthStatusFailed     MonthStatus = "failed"
)

type CalendarMonth struct {
	ID                    string      `db:"id"`
	ProjectID             string      `db:"project_id"`
	Month                 int         `db:"month"` // 1-12
	ReferencePhotoAssetID string      `db:"reference_photo_asset_id"`
	ReferenceImageURL     string      `db:"reference_image_url"`
	Prompt                string      `db:"prompt"`
	GeneratedImageURL     string      `db:"generated_image_url"`
	Status                MonthStatus `db:"status"`
	CreatedAt             time.Time   `db:"created_at"`
	UpdatedAt             time.Time   `db:"updated_at"`
}
