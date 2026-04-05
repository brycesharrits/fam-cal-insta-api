package domain

import "time"

type ProjectStatus string

const (
	ProjectStatusDraft      ProjectStatus = "draft"
	ProjectStatusGenerating ProjectStatus = "generating"
	ProjectStatusComplete   ProjectStatus = "complete"
	ProjectStatusOrdered    ProjectStatus = "ordered"
)

type CalendarProject struct {
	ID        string        `db:"id"`
	UserID    string        `db:"user_id"`
	Name      string        `db:"name"`
	Year      int           `db:"year"`
	Theme     string        `db:"theme"`
	Status    ProjectStatus `db:"status"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}
