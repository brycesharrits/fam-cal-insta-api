package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type OrderRepo struct {
	db *pgxpool.Pool
}

func NewOrderRepo(db *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	query := `
		INSERT INTO orders (user_id, calendar_id, partner, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query, o.UserID, o.CalendarID, o.Partner, o.Status).
		Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *OrderRepo) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT id, user_id, calendar_id, partner, status, partner_order_id, tracking_url, created_at, updated_at FROM orders WHERE id = $1`
	o := &domain.Order{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.UserID, &o.CalendarID, &o.Partner, &o.Status, &o.PartnerOrderID, &o.TrackingURL, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

func (r *OrderRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	query := `SELECT id, user_id, calendar_id, partner, status, partner_order_id, tracking_url, created_at, updated_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(&o.ID, &o.UserID, &o.CalendarID, &o.Partner, &o.Status, &o.PartnerOrderID, &o.TrackingURL, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepo) UpdateStatus(ctx context.Context, id, status, partnerOrderID, trackingURL string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status=$1, partner_order_id=$2, tracking_url=$3, updated_at=NOW() WHERE id=$4`,
		status, partnerOrderID, trackingURL, id,
	)
	return err
}
