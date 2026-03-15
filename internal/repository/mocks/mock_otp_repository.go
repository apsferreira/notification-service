package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
)

// MockOTPRepository implements the OTPRepository contract for unit tests.
// Each method delegates to an optional Fn field; if nil, returns zero values.
type MockOTPRepository struct {
	CreateFn             func(ctx context.Context, otp *model.OTPCode) error
	FindLatestByEmailFn  func(ctx context.Context, email string) (*model.OTPCode, error)
	IncrementAttemptsFn  func(ctx context.Context, id uuid.UUID) error
	DeleteByEmailFn      func(ctx context.Context, email string) error
	DeleteExpiredFn      func(ctx context.Context) error
	CountRecentByEmailFn func(ctx context.Context, email string, since time.Time) (int, error)
}

func (m *MockOTPRepository) Create(ctx context.Context, otp *model.OTPCode) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, otp)
	}
	return nil
}

func (m *MockOTPRepository) FindLatestByEmail(ctx context.Context, email string) (*model.OTPCode, error) {
	if m.FindLatestByEmailFn != nil {
		return m.FindLatestByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockOTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	if m.IncrementAttemptsFn != nil {
		return m.IncrementAttemptsFn(ctx, id)
	}
	return nil
}

func (m *MockOTPRepository) DeleteByEmail(ctx context.Context, email string) error {
	if m.DeleteByEmailFn != nil {
		return m.DeleteByEmailFn(ctx, email)
	}
	return nil
}

func (m *MockOTPRepository) DeleteExpired(ctx context.Context) error {
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn(ctx)
	}
	return nil
}

func (m *MockOTPRepository) CountRecentByEmail(ctx context.Context, email string, since time.Time) (int, error) {
	if m.CountRecentByEmailFn != nil {
		return m.CountRecentByEmailFn(ctx, email, since)
	}
	return 0, nil
}
