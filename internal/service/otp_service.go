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
	otpRepo            *repository.OTPRepository
	emailService       *EmailService
	telegramNotifier   *TelegramNotifier
	whatsappNotifier   WhatsAppNotifier
	expiryMinutes      int
	maxAttempts        int
	rateLimit          int
	rateWindow         time.Duration
}

func NewOTPService(
	otpRepo *repository.OTPRepository,
	emailService *EmailService,
	telegramNotifier *TelegramNotifier,
	whatsappNotifier WhatsAppNotifier,
	expiryMinutes, maxAttempts, rateLimit, rateWindowMinutes int,
) *OTPService {
	return &OTPService{
		otpRepo:          otpRepo,
		emailService:     emailService,
		telegramNotifier: telegramNotifier,
		whatsappNotifier: whatsappNotifier,
		expiryMinutes:    expiryMinutes,
		maxAttempts:      maxAttempts,
		rateLimit:        rateLimit,
		rateWindow:       time.Duration(rateWindowMinutes) * time.Minute,
	}
}

// GenerateAndSend creates a 6-digit OTP, stores it in DB, and sends via email/telegram.
// Mantido para compatibilidade retroativa — delega para GenerateAndSendChannel com canal "email".
func (s *OTPService) GenerateAndSend(ctx context.Context, email string) (time.Time, error) {
	return s.GenerateAndSendChannel(ctx, email, "", "email", "")
}

// GenerateAndSendChannel cria um OTP de 6 dígitos, armazena no DB e envia pelo canal informado.
// channel: "email" | "telegram" | "whatsapp"
// phone: número E.164 — obrigatório apenas quando channel == "whatsapp"
// telegramChatID: chat_id do usuário no Telegram — se vazio, usa o chat_id padrão configurado no notifier
func (s *OTPService) GenerateAndSendChannel(ctx context.Context, email, phone, channel, telegramChatID string) (time.Time, error) {
	if channel == "" {
		channel = "email"
	}

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
		Channel:   channel,
		Attempts:  0,
		ExpiresAt: time.Now().Add(time.Duration(s.expiryMinutes) * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return time.Time{}, fmt.Errorf("failed to store OTP: %w", err)
	}

	// Dispatch pelo canal solicitado
	switch channel {
	case "whatsapp":
		if phone == "" {
			return time.Time{}, fmt.Errorf("phone é obrigatório para o canal whatsapp")
		}
		if err := s.whatsappNotifier.SendOTP(phone, code); err != nil {
			return time.Time{}, fmt.Errorf("failed to send OTP via whatsapp to %s: %w", phone, err)
		}
	case "telegram":
		if err := s.telegramNotifier.SendOTP(telegramChatID, email, code); err != nil {
			return time.Time{}, fmt.Errorf("failed to send OTP via telegram for %s: %w", email, err)
		}
	default: // "email"
		if err := s.emailService.SendOTP(email, code); err != nil {
			// Se email falha, tenta telegram como fallback
			if telegramErr := s.telegramNotifier.SendOTP("", email, code); telegramErr != nil {
				return time.Time{}, fmt.Errorf("failed to send OTP via both email (%v) and telegram (%v)", err, telegramErr)
			}
		} else {
			// Entrega dupla via telegram para auth crítica
			_ = s.telegramNotifier.SendOTP("", email, code)
		}
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

// DeliverOnly envia um código OTP já gerado pelo auth-service, sem gerar nem armazenar.
// Responsabilidade: apenas entrega da mensagem pelo canal correto.
func (s *OTPService) DeliverOnly(ctx context.Context, email, code, channel, telegramChatID string) error {
	if channel == "" {
		channel = "email"
	}

	switch channel {
	case "telegram":
		if s.telegramNotifier == nil {
			return fmt.Errorf("telegram not configured")
		}
		chatID := telegramChatID
		if chatID == "" {
			return fmt.Errorf("telegram_chat_id is required")
		}
		return s.telegramNotifier.SendOTP(chatID, email, code)
	case "whatsapp":
		if s.whatsappNotifier == nil {
			return fmt.Errorf("whatsapp not configured")
		}
		return s.whatsappNotifier.SendOTP(email, code)
	default:
		return s.emailService.SendOTP(email, code)
	}
}