package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/repository"
)

type TemplateService struct {
	templateRepo *repository.TemplateRepository
}

func NewTemplateService(templateRepo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{templateRepo: templateRepo}
}

func (s *TemplateService) Create(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error) {
	return s.templateRepo.Create(ctx, req)
}

func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	return s.templateRepo.GetByID(ctx, id)
}

func (s *TemplateService) List(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error) {
	return s.templateRepo.List(ctx, limit, offset)
}

func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error) {
	return s.templateRepo.Update(ctx, id, req)
}
