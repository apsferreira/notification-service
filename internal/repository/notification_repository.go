package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	var variablesJSON []byte
	var err error
	if n.Variables != nil {
		variablesJSON, err = json.Marshal(n.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal variables: %w", err)
		}
	}

	query := `
		INSERT INTO notifications (id, type, recipient, subject, body, template_id, variables, status, attempts, error, created_at, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = r.db.Exec(ctx, query,
		n.ID, n.Type, n.Recipient, n.Subject, n.Body, n.TemplateID,
		variablesJSON, n.Status, n.Attempts, n.Error, n.CreatedAt, n.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return n, nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	query := `
		SELECT id, type, recipient, subject, body, template_id, variables, status, attempts, error, created_at, sent_at
		FROM notifications
		WHERE id = $1`

	return r.scanNotification(r.db.QueryRow(ctx, query, id))
}

func (r *NotificationRepository) List(ctx context.Context, filter *model.NotificationFilter) (*model.NotificationListResponse, error) {
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter.Type != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}
	if filter.Status != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}
	if filter.Recipient != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("recipient = $%d", argIndex))
		args = append(args, *filter.Recipient)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause)
	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Main query with pagination
	query := fmt.Sprintf(`
		SELECT id, type, recipient, subject, body, template_id, variables, status, attempts, error, created_at, sent_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		n, err := r.scanNotificationFromRows(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *n)
	}

	return &model.NotificationListResponse{
		Notifications: notifications,
		Total:         total,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	}, nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.NotificationStatus, attempts int, errMsg *string, sentAt *time.Time) error {
	query := `
		UPDATE notifications
		SET status = $1, attempts = $2, error = $3, sent_at = $4
		WHERE id = $5`

	_, err := r.db.Exec(ctx, query, status, attempts, errMsg, sentAt, id)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil
}

func (r *NotificationRepository) GetPendingRetries(ctx context.Context) ([]model.Notification, error) {
	query := `
		SELECT id, type, recipient, subject, body, template_id, variables, status, attempts, error, created_at, sent_at
		FROM notifications
		WHERE status IN ('pending', 'retrying') AND attempts < 3
		ORDER BY created_at ASC
		LIMIT 100`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending retries: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		n, err := r.scanNotificationFromRows(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *n)
	}

	return notifications, nil
}

func (r *NotificationRepository) scanNotification(row pgx.Row) (*model.Notification, error) {
	var n model.Notification
	var variablesJSON []byte

	err := row.Scan(
		&n.ID, &n.Type, &n.Recipient, &n.Subject, &n.Body,
		&n.TemplateID, &variablesJSON, &n.Status, &n.Attempts, &n.Error,
		&n.CreatedAt, &n.SentAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan notification: %w", err)
	}

	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &n.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
	}

	return &n, nil
}

func (r *NotificationRepository) scanNotificationFromRows(rows pgx.Rows) (*model.Notification, error) {
	var n model.Notification
	var variablesJSON []byte

	err := rows.Scan(
		&n.ID, &n.Type, &n.Recipient, &n.Subject, &n.Body,
		&n.TemplateID, &variablesJSON, &n.Status, &n.Attempts, &n.Error,
		&n.CreatedAt, &n.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan notification: %w", err)
	}

	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &n.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
	}

	return &n, nil
}
