package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type TokenRepo struct {
	db *pgxpool.Pool
}

func NewTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{db: db}
}

func (r *TokenRepo) RecordTransaction(ctx context.Context, tx *domain.TokenTransaction) error {
	query := `INSERT INTO token_transactions (user_id, amount, type, description) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRow(ctx, query, tx.UserID, tx.Amount, tx.Type, tx.Description).
		Scan(&tx.ID, &tx.CreatedAt)
}

func (r *TokenRepo) GetBalance(ctx context.Context, userID string) (int, error) {
	var balance int
	err := r.db.QueryRow(ctx, `SELECT token_balance FROM users WHERE id = $1`, userID).Scan(&balance)
	return balance, err
}

func (r *TokenRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.TokenTransaction, error) {
	query := `SELECT id, user_id, amount, type, description, created_at FROM token_transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.TokenTransaction
	for rows.Next() {
		tx := &domain.TokenTransaction{}
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Type, &tx.Description, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

// DeductAtomic checks balance, deducts amount, and records transaction in one DB transaction.
func (r *TokenRepo) DeductAtomic(ctx context.Context, userID string, amount int, description string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var balance int
	err = tx.QueryRow(ctx, `SELECT token_balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balance)
	if err != nil {
		return err
	}
	if balance < amount {
		return errors.New("insufficient token balance")
	}

	_, err = tx.Exec(ctx, `UPDATE users SET token_balance = token_balance - $1, updated_at = NOW() WHERE id = $2`, amount, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO token_transactions (user_id, amount, type, description) VALUES ($1, $2, 'spend', $3)`,
		userID, -amount, fmt.Sprintf("spent %d tokens: %s", amount, description),
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
