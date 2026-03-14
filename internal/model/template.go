package model

import (
	"time"

	"github.com/google/uuid"
)

// Template represents a notification template
type Template struct {
	ID              uuid.UUID        `json:"id"`
	Name            string           `json:"name"`
	Type            NotificationType `json:"type"`
	SubjectTemplate string           `json:"subject_template"`
	BodyTemplate    string           `json:"body_template"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// CreateTemplateRequest represents the request to create a template
type CreateTemplateRequest struct {
	Name            string           `json:"name" validate:"required,min=1,max=255"`
	Type            NotificationType `json:"type" validate:"required,oneof=email sms"`
	SubjectTemplate string           `json:"subject_template" validate:"required_if=Type email"`
	BodyTemplate    string           `json:"body_template" validate:"required"`
}

// UpdateTemplateRequest represents the request to update a template
type UpdateTemplateRequest struct {
	Name            *string           `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Type            *NotificationType `json:"type,omitempty" validate:"omitempty,oneof=email sms"`
	SubjectTemplate *string           `json:"subject_template,omitempty"`
	BodyTemplate    *string           `json:"body_template,omitempty"`
}

// TemplateListResponse represents the response for listing templates
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
	Total     int        `json:"total"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}
