package domain

import "time"

type User struct {
	ID           string    `db:"id"`
	AppleUserID  string    `db:"apple_user_id"`
	Email        string    `db:"email"`
	TokenBalance int       `db:"token_balance"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
