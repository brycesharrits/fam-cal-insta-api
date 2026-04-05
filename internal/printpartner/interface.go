package printpartner

import (
	"context"
	"time"
)

type Address struct {
	Name       string
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

type MonthImage struct {
	Month    int    // 1-12
	ImageURL string // S3 URL to the generated image
}

type CalendarPayload struct {
	ProjectID    string
	Year         int
	MonthImages  []MonthImage // exactly 12
	ShippingAddr Address
}

type OrderResult struct {
	PartnerOrderID string
	EstDelivery    time.Time
	TrackingURL    string
}

// PrintPartner is the interface all print fulfillment partners must implement.
// Swap Printful/Chatbooks/Snapfish by implementing this interface.
type PrintPartner interface {
	// SubmitOrder sends a completed calendar to the print partner for fulfillment.
	SubmitOrder(ctx context.Context, payload CalendarPayload) (*OrderResult, error)

	// GetOrderStatus retrieves the current fulfillment status from the partner.
	GetOrderStatus(ctx context.Context, partnerOrderID string) (string, error)

	// PartnerName returns a human-readable name for logging.
	PartnerName() string
}
