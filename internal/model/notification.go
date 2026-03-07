package model

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending  NotificationStatus = "pending"
	NotificationStatusSent     NotificationStatus = "sent"
	NotificationStatusFailed   NotificationStatus = "failed"
	NotificationStatusRetrying NotificationStatus = "retrying"
)

// Notification represents a notification record in the system
type Notification struct {
	ID         uuid.UUID              `json:"id"`
	Type       NotificationType       `json:"type"`
	Recipient  string                 `json:"recipient"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body"`
	TemplateID *uuid.UUID             `json:"template_id,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	Status     NotificationStatus     `json:"status"`
	Attempts   int                    `json:"attempts"`
	Error      *string                `json:"error,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	SentAt     *time.Time             `json:"sent_at,omitempty"`
}

// SendNotificationRequest represents the request to send a notification
type SendNotificationRequest struct {
	Type       NotificationType       `json:"type" validate:"required,oneof=email sms"`
	Recipient  string                 `json:"recipient" validate:"required"`
	Subject    string                 `json:"subject" validate:"required_if=Type email"`
	Body       string                 `json:"body,omitempty"`
	TemplateID *uuid.UUID             `json:"template_id,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

// NotificationFilter represents query filters for listing notifications
type NotificationFilter struct {
	Type      *NotificationType   `json:"type,omitempty"`
	Status    *NotificationStatus `json:"status,omitempty"`
	Recipient *string             `json:"recipient,omitempty"`
	Limit     int                 `json:"limit,omitempty"`
	Offset    int                 `json:"offset,omitempty"`
}

// NotificationListResponse represents the response for listing notifications
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Limit         int            `json:"limit"`
	Offset        int            `json:"offset"`
}
