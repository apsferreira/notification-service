package service

import (
	"fmt"
	"log"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client    *resend.Client
	fromEmail string
	isDev     bool
}

func NewEmailService(apiKey, fromEmail, env string) *EmailService {
	var client *resend.Client
	if apiKey != "" {
		client = resend.NewClient(apiKey)
	}
	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
		isDev:     env == "development",
	}
}

func (s *EmailService) SendOTP(toEmail, code string) error {
	if s.isDev {
		log.Printf("[DEV] OTP for %s: %s", toEmail, code)
	}

	if s.client == nil {
		if s.isDev {
			return nil
		}
		return fmt.Errorf("email service not configured: RESEND_API_KEY is missing")
	}

	html := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 400px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #333;">Your verification code</h2>
			<p style="font-size: 36px; font-weight: bold; letter-spacing: 8px; color: #2563eb; margin: 20px 0;">%s</p>
			<p style="color: #666;">This code expires in 10 minutes.</p>
			<p style="color: #999; font-size: 12px;">If you didn't request this code, you can safely ignore this email.</p>
		</div>
	`, code)

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{toEmail},
		Subject: "Your login code",
		Html:    html,
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		if s.isDev {
			log.Printf("[DEV] Resend error (OTP still valid — use code from logs): %v", err)
			return nil
		}
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}