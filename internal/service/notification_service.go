package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/repository"
)

const maxRetries = 3

type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	templateRepo     *repository.TemplateRepository
	resendAPIKey     string
	fromEmail        string
	httpClient       *http.Client
}

func NewNotificationService(
	notificationRepo *repository.NotificationRepository,
	templateRepo *repository.TemplateRepository,
	resendAPIKey string,
	fromEmail string,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		templateRepo:     templateRepo,
		resendAPIKey:     resendAPIKey,
		fromEmail:        fromEmail,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send creates a notification record and attempts to send it
func (s *NotificationService) Send(ctx context.Context, req *model.SendNotificationRequest) (*model.Notification, error) {
	subject := req.Subject
	body := req.Body

	// If a template is provided, render it
	if req.TemplateID != nil {
		tmpl, err := s.templateRepo.GetByID(ctx, *req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("failed to get template: %w", err)
		}
		if tmpl == nil {
			return nil, fmt.Errorf("template not found: %s", req.TemplateID.String())
		}

		subject = renderTemplate(tmpl.SubjectTemplate, req.Variables)
		body = renderTemplate(tmpl.BodyTemplate, req.Variables)
	}

	notification := &model.Notification{
		ID:         uuid.New(),
		Type:       req.Type,
		Recipient:  req.Recipient,
		Subject:    subject,
		Body:       body,
		TemplateID: req.TemplateID,
		Variables:  req.Variables,
		Status:     model.NotificationStatusPending,
		Attempts:   0,
		CreatedAt:  time.Now(),
	}

	// Save to database
	notification, err := s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Attempt to send
	s.attemptSend(ctx, notification)

	// Re-fetch to get updated status
	updated, err := s.notificationRepo.GetByID(ctx, notification.ID)
	if err != nil {
		return notification, nil // Return original if re-fetch fails
	}

	return updated, nil
}

func (s *NotificationService) attemptSend(ctx context.Context, n *model.Notification) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		n.Attempts = attempt

		switch n.Type {
		case model.NotificationTypeEmail:
			lastErr = s.sendEmail(ctx, n)
		case model.NotificationTypeSMS:
			lastErr = fmt.Errorf("SMS sending not yet implemented")
		default:
			lastErr = fmt.Errorf("unknown notification type: %s", n.Type)
		}

		if lastErr == nil {
			// Success
			now := time.Now()
			_ = s.notificationRepo.UpdateStatus(ctx, n.ID, model.NotificationStatusSent, attempt, nil, &now)
			return
		}

		log.Printf("[notification] Attempt %d/%d failed for %s: %v", attempt, maxRetries, n.ID, lastErr)

		if attempt < maxRetries {
			errMsg := lastErr.Error()
			_ = s.notificationRepo.UpdateStatus(ctx, n.ID, model.NotificationStatusRetrying, attempt, &errMsg, nil)
			// Brief backoff between retries
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	// All retries exhausted
	errMsg := lastErr.Error()
	_ = s.notificationRepo.UpdateStatus(ctx, n.ID, model.NotificationStatusFailed, maxRetries, &errMsg, nil)
}

func (s *NotificationService) sendEmail(ctx context.Context, n *model.Notification) error {
	payload := map[string]interface{}{
		"from":    s.fromEmail,
		"to":      []string{n.Recipient},
		"subject": n.Subject,
		"html":    n.Body,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.resendAPIKey))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// renderTemplate replaces {{variable_name}} placeholders with values from the variables map
func renderTemplate(template string, variables map[string]interface{}) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result
}

// GetNotification retrieves a notification by ID
func (s *NotificationService) GetNotification(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	return s.notificationRepo.GetByID(ctx, id)
}

// ListNotifications lists notifications with filters
func (s *NotificationService) ListNotifications(ctx context.Context, filter *model.NotificationFilter) (*model.NotificationListResponse, error) {
	return s.notificationRepo.List(ctx, filter)
}

// RetryPending retries all pending/retrying notifications (can be called by a cron/background worker)
func (s *NotificationService) RetryPending(ctx context.Context) error {
	notifications, err := s.notificationRepo.GetPendingRetries(ctx)
	if err != nil {
		return err
	}

	for i := range notifications {
		s.attemptSend(ctx, &notifications[i])
	}

	return nil
}
