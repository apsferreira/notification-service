package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TemplateRepository struct {
	db *pgxpool.Pool
}

func NewTemplateRepository(db *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, req *model.CreateTemplateRequest) (*model.Template, error) {
	t := &model.Template{
		ID:              uuid.New(),
		Name:            req.Name,
		Type:            req.Type,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	query := `
		INSERT INTO templates (id, name, type, subject_template, body_template, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(ctx, query,
		t.ID, t.Name, t.Type, t.SubjectTemplate, t.BodyTemplate, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return t, nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	query := `
		SELECT id, name, type, subject_template, body_template, created_at, updated_at
		FROM templates
		WHERE id = $1`

	var t model.Template
	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Type, &t.SubjectTemplate, &t.BodyTemplate, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return &t, nil
}

func (r *TemplateRepository) List(ctx context.Context, limit, offset int) (*model.TemplateListResponse, error) {
	// Count total
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM templates").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	query := `
		SELECT id, name, type, subject_template, body_template, created_at, updated_at
		FROM templates
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var t model.Template
		err := rows.Scan(
			&t.ID, &t.Name, &t.Type, &t.SubjectTemplate, &t.BodyTemplate, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, t)
	}

	return &model.TemplateListResponse{
		Templates: templates,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

func (r *TemplateRepository) Update(ctx context.Context, id uuid.UUID, req *model.UpdateTemplateRequest) (*model.Template, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Type != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *req.Type)
		argIndex++
	}
	if req.SubjectTemplate != nil {
		updates = append(updates, fmt.Sprintf("subject_template = $%d", argIndex))
		args = append(args, *req.SubjectTemplate)
		argIndex++
	}
	if req.BodyTemplate != nil {
		updates = append(updates, fmt.Sprintf("body_template = $%d", argIndex))
		args = append(args, *req.BodyTemplate)
		argIndex++
	}

	if len(updates) == 0 {
		return existing, nil
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf("UPDATE templates SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM templates WHERE id = $1`, id)
	return err
}
