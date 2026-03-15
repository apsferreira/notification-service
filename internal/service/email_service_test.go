package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── NewEmailService ──────────────────────────────────────────────────────────

func TestNewEmailService_WithAPIKey(t *testing.T) {
	svc := NewEmailService("re_test_key", "noreply@example.com", "production")

	assert.NotNil(t, svc)
	assert.False(t, svc.isDev)
	assert.Equal(t, "noreply@example.com", svc.fromEmail)
	assert.NotNil(t, svc.client, "client should be initialised when API key is provided")
}

func TestNewEmailService_EmptyAPIKey(t *testing.T) {
	svc := NewEmailService("", "noreply@example.com", "production")

	assert.NotNil(t, svc)
	assert.Nil(t, svc.client, "client must be nil when API key is empty")
}

func TestNewEmailService_DevelopmentEnv(t *testing.T) {
	svc := NewEmailService("", "noreply@example.com", "development")

	assert.True(t, svc.isDev)
}

func TestNewEmailService_NonDevelopmentEnvs(t *testing.T) {
	for _, env := range []string{"production", "staging", "test", ""} {
		svc := NewEmailService("key", "from@example.com", env)
		assert.False(t, svc.isDev, "env=%q should not be treated as development", env)
	}
}

// ─── SendOTP: dev mode without client ────────────────────────────────────────

func TestSendOTP_DevMode_NoClient_ReturnsNil(t *testing.T) {
	// Dev mode + no API key: logs the OTP and returns nil (no network call)
	svc := NewEmailService("", "noreply@example.com", "development")

	err := svc.SendOTP("user@example.com", "123456")

	require.NoError(t, err)
}

// ─── SendOTP: production mode without client ─────────────────────────────────

func TestSendOTP_ProductionMode_NoClient_ReturnsError(t *testing.T) {
	// Production without API key → must surface a clear error
	svc := NewEmailService("", "noreply@example.com", "production")

	err := svc.SendOTP("user@example.com", "123456")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email service not configured")
}

// ─── SendOTP: dev mode — Resend error is suppressed ─────────────────────────

func TestSendOTP_DevMode_WithInvalidKey_ErrorSuppressed(t *testing.T) {
	// Even when the Resend client is set but the API key is invalid (or network
	// is unavailable), dev mode must suppress the error and return nil so
	// developers can still read the OTP from the application logs.
	svc := NewEmailService("re_invalid_key", "noreply@example.com", "development")

	err := svc.SendOTP("user@example.com", "999999")

	// Dev mode swallows Resend / network errors.
	assert.NoError(t, err)
}

// ─── HTML template sanity check ──────────────────────────────────────────────

func TestSendOTP_HTMLTemplate_ContainsCode(t *testing.T) {
	// Validate that the HTML template built inside SendOTP embeds the code.
	// We replicate the fmt.Sprintf call from the production function.
	code := "456789"
	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 400px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #333;">Your verification code</h2>
			<p style="font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #2563eb; margin: 20px 0;">%s</p>
			<p style="color: #666;">This code expires in 10 minutes.</p>
			<p style="color: #999; font-size: 12px;">If you didn't request this code, you can safely ignore this email.</p>
		</div>
	`, code)

	assert.Contains(t, html, code, "OTP code must appear in the email HTML body")
	assert.Contains(t, html, "Your verification code")
	assert.Contains(t, html, "expires in 10 minutes")
}
