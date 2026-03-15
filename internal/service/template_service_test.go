package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/repository/mocks"
)

func TestTemplateCreate_Success(t *testing.T) {
	mockRepo := &mocks.MockTemplateRepository{
		CreateFn: func(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error) {
			return &model.Template{
				ID:              uuid.New(),
				Name:            req.Name,
				Type:            req.Type,
				SubjectTemplate: req.SubjectTemplate,
				BodyTemplate:    req.BodyTemplate,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}, nil
		},
	}

	service := NewTemplateService(mockRepo)

	req := &model.CreateTemplateRequest{
		Name:            "welcome",
		Type:            model.NotificationTypeEmail,
		SubjectTemplate: "Welcome!",
		BodyTemplate:    "Hello {{.name}}",
	}

	tmpl, err := service.Create(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tmpl.Name != "welcome" {
		t.Errorf("expected name 'welcome', got: %s", tmpl.Name)
	}
}

func TestTemplateCreate_RepositoryError(t *testing.T) {
	mockRepo := &mocks.MockTemplateRepository{
		CreateFn: func(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error) {
			return nil, errors.New("database error")
		},
	}

	service := NewTemplateService(mockRepo)

	req := &model.CreateTemplateRequest{
		Name:            "welcome",
		Type:            model.NotificationTypeEmail,
		SubjectTemplate: "Welcome!",
		BodyTemplate:    "Hello",
	}

	_, err := service.Create(context.Background(), req)

	if err == nil {
		t.Fatal("expected error when repository fails, got nil")
	}
}

func TestTemplateGetByID_Success(t *testing.T) {
	templateID := uuid.New()

	mockRepo := &mocks.MockTemplateRepository{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
			return &model.Template{
				ID:              templateID,
				Name:            "welcome",
				Type:            model.NotificationTypeEmail,
				SubjectTemplate: "Welcome!",
				BodyTemplate:    "Hello {{.name}}",
			}, nil
		},
	}

	service := NewTemplateService(mockRepo)

	tmpl, err := service.GetByID(context.Background(), templateID)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tmpl.ID != templateID {
		t.Errorf("expected template ID %s, got: %s", templateID, tmpl.ID)
	}
}

func TestTemplateList_Success(t *testing.T) {
	mockRepo := &mocks.MockTemplateRepository{
		ListFn: func(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error) {
			return &model.TemplateListResponse{
				Templates: []model.Template{
					{ID: uuid.New(), Name: "welcome"},
					{ID: uuid.New(), Name: "reset-password"},
				},
				Total: 2,
			}, nil
		},
	}

	service := NewTemplateService(mockRepo)

	result, err := service.List(context.Background(), 10, 0)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("expected total 2, got: %d", result.Total)
	}

	if len(result.Templates) != 2 {
		t.Errorf("expected 2 templates, got: %d", len(result.Templates))
	}
}

func TestTemplateUpdate_Success(t *testing.T) {
	templateID := uuid.New()

	mockRepo := &mocks.MockTemplateRepository{
		UpdateFn: func(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error) {
			return &model.Template{
				ID:              id,
				Name:            *req.Name,
				SubjectTemplate: *req.SubjectTemplate,
				BodyTemplate:    *req.BodyTemplate,
				UpdatedAt:       time.Now(),
			}, nil
		},
	}

	service := NewTemplateService(mockRepo)

	name := "updated"
	subject := "Updated Subject"
	body := "Updated Body"
	req := &model.UpdateTemplateRequest{
		Name:            &name,
		SubjectTemplate: &subject,
		BodyTemplate:    &body,
	}

	tmpl, err := service.Update(context.Background(), templateID, req)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if tmpl.Name != "updated" {
		t.Errorf("expected name 'updated', got: %s", tmpl.Name)
	}
}
