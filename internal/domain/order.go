package domain

import "time"

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusFailed     OrderStatus = "failed"
)

type Order struct {
	ID             string      `db:"id"`
	UserID         string      `db:"user_id"`
	CalendarID     string      `db:"calendar_id"`
	Partner        string      `db:"partner"`
	Status         OrderStatus `db:"status"`
	PartnerOrderID string      `db:"partner_order_id"`
	TrackingURL    string      `db:"tracking_url"`
	CreatedAt      time.Time   `db:"created_at"`
	UpdatedAt      time.Time   `db:"updated_at"`
}
