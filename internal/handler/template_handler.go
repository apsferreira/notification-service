package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/service"
)

type TemplateHandler struct {
	templateSvc *service.TemplateService
	validator   *validator.Validate
}

func NewTemplateHandler(templateSvc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateSvc: templateSvc,
		validator:   validator.New(),
	}
}

// POST /templates
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "validation failed",
			"errors": validationErrors,
		})
		return
	}

	template, err := h.templateSvc.Create(r.Context(), &req)
	if err != nil {
		http.Error(w, `{"error": "failed to create template"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// GET /templates
func (h *TemplateHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	response, err := h.templateSvc.List(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, `{"error": "failed to list templates"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /templates/{id}
func (h *TemplateHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, `{"error": "invalid template id"}`, http.StatusBadRequest)
		return
	}

	template, err := h.templateSvc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error": "failed to get template"}`, http.StatusInternalServerError)
		return
	}

	if template == nil {
		http.Error(w, `{"error": "template not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// PUT /templates/{id}
func (h *TemplateHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, `{"error": "invalid template id"}`, http.StatusBadRequest)
		return
	}

	var req model.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		validationErrors := make([]string, 0)
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "validation failed",
			"errors": validationErrors,
		})
		return
	}

	template, err := h.templateSvc.Update(r.Context(), id, &req)
	if err != nil {
		http.Error(w, `{"error": "failed to update template"}`, http.StatusInternalServerError)
		return
	}

	if template == nil {
		http.Error(w, `{"error": "template not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}
