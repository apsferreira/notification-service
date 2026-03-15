package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
)

// NotificationRepositoryInterface defines the contract for notification repository operations
type NotificationRepositoryInterface interface {
	Create(ctx context.Context, n *model.Notification) (*model.Notification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	List(ctx context.Context, filter *model.NotificationFilter) (*model.NotificationListResponse, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, attempts int, errMsg *string, sentAt *time.Time) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	GetPendingRetries(ctx context.Context) ([]model.Notification, error)
}

// TemplateRepositoryInterface defines the contract for template repository operations
type TemplateRepositoryInterface interface {
	Create(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error)
	List(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
