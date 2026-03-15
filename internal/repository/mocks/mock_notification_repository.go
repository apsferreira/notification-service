package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
)

type MockNotificationRepository struct {
	CreateFn             func(ctx context.Context, n *model.Notification) (*model.Notification, error)
	GetByIDFn            func(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	ListFn               func(ctx context.Context, filter *model.NotificationFilter) (*model.NotificationListResponse, error)
	UpdateStatusFn       func(ctx context.Context, id uuid.UUID, status model.NotificationStatus, attempts int, errMsg *string, sentAt *time.Time) error
	IncrementAttemptsFn  func(ctx context.Context, id uuid.UUID) error
	GetPendingRetriesFn  func(ctx context.Context) ([]model.Notification, error)
}

func (m *MockNotificationRepository) Create(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, n)
	}
	return nil, nil
}

func (m *MockNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockNotificationRepository) List(ctx context.Context, filter *model.NotificationFilter) (*model.NotificationListResponse, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return nil, nil
}

func (m *MockNotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, attempts int, errMsg *string, sentAt *time.Time) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status, attempts, errMsg, sentAt)
	}
	return nil
}

func (m *MockNotificationRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	if m.IncrementAttemptsFn != nil {
		return m.IncrementAttemptsFn(ctx, id)
	}
	return nil
}

func (m *MockNotificationRepository) GetPendingRetries(ctx context.Context) ([]model.Notification, error) {
	if m.GetPendingRetriesFn != nil {
		return m.GetPendingRetriesFn(ctx)
	}
	return nil, nil
}
