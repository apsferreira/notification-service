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

type NotificationHandler struct {
	notificationSvc *service.NotificationService
	validator       *validator.Validate
}

func NewNotificationHandler(notificationSvc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationSvc: notificationSvc,
		validator:       validator.New(),
	}
}

// POST /notifications/send
func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req model.SendNotificationRequest
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

	notification, err := h.notificationSvc.Send(r.Context(), &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

// GET /notifications
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	filter := &model.NotificationFilter{}

	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		t := model.NotificationType(typeParam)
		filter.Type = &t
	}
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		s := model.NotificationStatus(statusParam)
		filter.Status = &s
	}
	if recipientParam := r.URL.Query().Get("recipient"); recipientParam != "" {
		filter.Recipient = &recipientParam
	}

	limit := 20
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	filter.Limit = limit

	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}
	filter.Offset = offset

	response, err := h.notificationSvc.ListNotifications(r.Context(), filter)
	if err != nil {
		http.Error(w, `{"error": "failed to list notifications"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GET /notifications/{id}
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, `{"error": "invalid notification id"}`, http.StatusBadRequest)
		return
	}

	notification, err := h.notificationSvc.GetNotification(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error": "failed to get notification"}`, http.StatusInternalServerError)
		return
	}

	if notification == nil {
		http.Error(w, `{"error": "notification not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}
