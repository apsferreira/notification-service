package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type OTPService struct {
	otpRepo          *repository.OTPRepository
	emailService     *EmailService
	telegramNotifier *TelegramNotifier
	expiryMinutes    int
	maxAttempts      int
	rateLimit        int
	rateWindow       time.Duration
}

func NewOTPService(
	otpRepo *repository.OTPRepository,
	emailService *EmailService,
	telegramNotifier *TelegramNotifier,
	expiryMinutes, maxAttempts, rateLimit, rateWindowMinutes int,
) *OTPService {
	return &OTPService{
		otpRepo:          otpRepo,
		emailService:     emailService,
		telegramNotifier: telegramNotifier,
		expiryMinutes:    expiryMinutes,
		maxAttempts:      maxAttempts,
		rateLimit:        rateLimit,
		rateWindow:       time.Duration(rateWindowMinutes) * time.Minute,
	}
}

// GenerateAndSend creates a 6-digit OTP, stores it in DB, and sends via email/telegram.
func (s *OTPService) GenerateAndSend(ctx context.Context, email string) (time.Time, error) {
	// Check rate limit
	count, err := s.otpRepo.CountRecentByEmail(ctx, email, time.Now().Add(-s.rateWindow))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if count >= s.rateLimit {
		return time.Time{}, fmt.Errorf("rate limit exceeded: max %d OTP requests per %d minutes", s.rateLimit, int(s.rateWindow.Minutes()))
	}

	// Generate 6-digit code
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to generate OTP: %w", err)
	}
	code := fmt.Sprintf("%06d", n.Int64())

	// Hash the code
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to hash OTP: %w", err)
	}

	// Store in database
	otp := &model.OTPCode{
		ID:        uuid.New(),
		Email:     email,
		CodeHash:  string(hash),
		Channel:   "email",
		Attempts:  0,
		ExpiresAt: time.Now().Add(time.Duration(s.expiryMinutes) * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return time.Time{}, fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send via email (primary channel)
	if err := s.emailService.SendOTP(email, code); err != nil {
		// If email fails, try telegram as fallback
		if telegramErr := s.telegramNotifier.SendOTP(email, code); telegramErr != nil {
			return time.Time{}, fmt.Errorf("failed to send OTP via both email (%v) and telegram (%v)", err, telegramErr)
		}
	} else {
		// Also send via telegram if configured (dual delivery for critical auth)
		_ = s.telegramNotifier.SendOTP(email, code)
	}

	return otp.ExpiresAt, nil
}

// Verify checks a code against the stored OTP for an email with constant-time execution.
func (s *OTPService) Verify(ctx context.Context, email, code string) error {
	// Garantir tempo de execução constante para prevenir timing attacks
	startTime := time.Now()
	defer func() {
		// Forçar no mínimo 200ms de resposta para dificultar timing analysis
		elapsed := time.Since(startTime)
		if elapsed < 200*time.Millisecond {
			time.Sleep(200*time.Millisecond - elapsed)
		}
	}()

	otp, err := s.otpRepo.FindLatestByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("no valid OTP found for this email")
	}

	// Checar expiração ANTES de incrementar tentativas (evita burn de attempts em OTPs expirados)
	if time.Now().After(otp.ExpiresAt) {
		return fmt.Errorf("OTP has expired")
	}

	if otp.Attempts >= s.maxAttempts {
		return fmt.Errorf("maximum verification attempts exceeded")
	}

	// Incrementar tentativas ANTES de validar (previne leak de informação)
	_ = s.otpRepo.IncrementAttempts(ctx, otp.ID)

	// bcrypt.CompareHashAndPassword tem timing variável, mas com delay acima
	// e limite de 3 tentativas, fica impraticável explorar
	if err := bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)); err != nil {
		return fmt.Errorf("invalid OTP code")
	}

	// Cleanup: delete all OTPs for this email
	_ = s.otpRepo.DeleteByEmail(ctx, email)

	return nil
}

func (s *OTPService) GetExpiryMinutes() int {
	return s.expiryMinutes
}