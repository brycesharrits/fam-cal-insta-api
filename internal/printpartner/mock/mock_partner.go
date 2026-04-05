package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/brycesharrits/fam-cal-insta/internal/printpartner"
)

// MockPartner is a stub print partner used until a real partner is integrated.
type MockPartner struct{}

func New() *MockPartner {
	return &MockPartner{}
}

func (m *MockPartner) SubmitOrder(_ context.Context, payload printpartner.CalendarPayload) (*printpartner.OrderResult, error) {
	return &printpartner.OrderResult{
		PartnerOrderID: fmt.Sprintf("MOCK-%s-%d", payload.ProjectID[:8], time.Now().Unix()),
		EstDelivery:    time.Now().Add(14 * 24 * time.Hour),
		TrackingURL:    "",
	}, nil
}

func (m *MockPartner) GetOrderStatus(_ context.Context, partnerOrderID string) (string, error) {
	return "processing", nil
}

func (m *MockPartner) PartnerName() string {
	return "mock"
}
