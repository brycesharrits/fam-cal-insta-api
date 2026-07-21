package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (provider, provider_user_id, email, token_balance)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query, user.Provider, user.ProviderUserID, user.Email, user.TokenBalance).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, provider, provider_user_id, email, token_balance, created_at, updated_at FROM users WHERE id = $1`
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Provider, &user.ProviderUserID, &user.Email, &user.TokenBalance, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepo) FindByProviderID(ctx context.Context, provider, providerUserID string) (*domain.User, error) {
	query := `SELECT id, provider, provider_user_id, email, token_balance, created_at, updated_at FROM users WHERE provider = $1 AND provider_user_id = $2`
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, provider, providerUserID).Scan(
		&user.ID, &user.Provider, &user.ProviderUserID, &user.Email, &user.TokenBalance, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (r *UserRepo) UpdateTokenBalance(ctx context.Context, userID string, delta int) error {
	query := `UPDATE users SET token_balance = token_balance + $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.db.Exec(ctx, query, delta, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user %s not found", userID)
	}
	return nil
}
