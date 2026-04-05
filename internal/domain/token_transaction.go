package domain

import "time"

type TransactionType string

const (
	TransactionTypePurchase TransactionType = "purchase"
	TransactionTypeSpend    TransactionType = "spend"
	TransactionTypeRefund   TransactionType = "refund"
)

type TokenTransaction struct {
	ID          string          `db:"id"`
	UserID      string          `db:"user_id"`
	Amount      int             `db:"amount"`
	Type        TransactionType `db:"type"`
	Description string          `db:"description"`
	CreatedAt   time.Time       `db:"created_at"`
}
