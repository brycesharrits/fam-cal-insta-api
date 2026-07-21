package domain

import "time"

const (
	ProviderApple  = "apple"
	ProviderGoogle = "google"
)

type User struct {
	ID             string    `db:"id"`
	Provider       string    `db:"provider"`
	ProviderUserID string    `db:"provider_user_id"`
	Email          string    `db:"email"`
	TokenBalance   int       `db:"token_balance"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
