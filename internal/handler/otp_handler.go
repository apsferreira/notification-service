package handler

import (
	"encoding/json"
	"net/http"

	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/service"
)

type OTPHandler struct {
	otpService *service.OTPService
}

func NewOTPHandler(otpService *service.OTPService) *OTPHandler {
	return &OTPHandler{
		otpService: otpService,
	}
}

// SendOTP handles POST /otp/send
func (h *OTPHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req model.SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	if req.Channel == "whatsapp" && req.Phone == "" {
		http.Error(w, "phone is required for whatsapp channel", http.StatusBadRequest)
		return
	}

	if req.Channel == "telegram" && req.TelegramChatID == "" {
		http.Error(w, "telegram_chat_id is required for telegram channel", http.StatusBadRequest)
		return
	}

	expiresAt, err := h.otpService.GenerateAndSendChannel(r.Context(), req.Email, req.Phone, req.Channel, req.TelegramChatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := model.OTPResponse{
		Message:   "OTP sent successfully",
		ExpiresAt: expiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// VerifyOTP handles POST /otp/verify
func (h *OTPHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Code == "" {
		http.Error(w, "Email and code are required", http.StatusBadRequest)
		return
	}

	err := h.otpService.Verify(r.Context(), req.Email, req.Code)
	if err != nil {
		response := model.VerifyOTPResponse{
			Valid:   false,
			Message: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := model.VerifyOTPResponse{
		Valid:   true,
		Message: "OTP verified successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}