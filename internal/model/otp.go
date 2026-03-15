package model

import (
	"time"

	"github.com/google/uuid"
)

// OTPCode represents a one-time password stored in the database
type OTPCode struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CodeHash  string    `json:"-"`
	Channel   string    `json:"channel"`
	Attempts  int       `json:"attempts"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SendOTPRequest represents the request to send an OTP
type SendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// VerifyOTPRequest represents the request to verify an OTP
type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

// OTPResponse represents the response after sending an OTP
type OTPResponse struct {
	Message   string    `json:"message"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VerifyOTPResponse represents the response after verifying an OTP
type VerifyOTPResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}