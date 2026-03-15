package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
)

type MockTemplateRepository struct {
	CreateFn  func(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error)
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*model.Template, error)
	ListFn    func(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error)
	UpdateFn  func(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error)
	DeleteFn  func(ctx context.Context, id uuid.UUID) error
}

func (m *MockTemplateRepository) Create(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req)
	}
	return nil, nil
}

func (m *MockTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockTemplateRepository) List(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *MockTemplateRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, id, req)
	}
	return nil, nil
}

func (m *MockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
