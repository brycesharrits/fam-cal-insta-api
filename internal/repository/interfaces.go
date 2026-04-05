package repository

import (
	"context"

	"github.com/brycesharrits/fam-cal-insta/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByAppleUserID(ctx context.Context, appleUserID string) (*domain.User, error)
	UpdateTokenBalance(ctx context.Context, userID string, delta int) error
}

type ProjectRepository interface {
	Create(ctx context.Context, p *domain.CalendarProject) error
	FindByID(ctx context.Context, id string) (*domain.CalendarProject, error)
	FindByUserID(ctx context.Context, userID string) ([]*domain.CalendarProject, error)
	Update(ctx context.Context, p *domain.CalendarProject) error
	Delete(ctx context.Context, id string) error
}

type MonthRepository interface {
	Upsert(ctx context.Context, m *domain.CalendarMonth) error
	FindByProjectID(ctx context.Context, projectID string) ([]*domain.CalendarMonth, error)
	FindByID(ctx context.Context, id string) (*domain.CalendarMonth, error)
	UpdateGeneratedImage(ctx context.Context, id, imageURL string, status domain.MonthStatus) error
	UpdatePromptAndRef(ctx context.Context, id, prompt, refImageURL string) error
}

type GenerationJobRepository interface {
	Create(ctx context.Context, job *domain.GenerationJob) error
	FindByID(ctx context.Context, id string) (*domain.GenerationJob, error)
	FindByReplicatePredictionID(ctx context.Context, predictionID string) (*domain.GenerationJob, error)
	FindByCalendarID(ctx context.Context, calendarID string) ([]*domain.GenerationJob, error)
	UpdateStatus(ctx context.Context, id string, status domain.JobStatus, resultURL, errMsg string) error
	UpdatePredictionID(ctx context.Context, id, predictionID string) error
}

type TokenRepository interface {
	RecordTransaction(ctx context.Context, tx *domain.TokenTransaction) error
	GetBalance(ctx context.Context, userID string) (int, error)
	FindByUserID(ctx context.Context, userID string) ([]*domain.TokenTransaction, error)
	// DeductAtomic checks balance, deducts amount, and records transaction in one DB transaction.
	// Returns error if balance is insufficient.
	DeductAtomic(ctx context.Context, userID string, amount int, description string) error
}

type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	FindByID(ctx context.Context, id string) (*domain.Order, error)
	FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status, partnerOrderID, trackingURL string) error
}
