package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// TelegramNotifier delivers OTP codes via Telegram Bot API.
// Used as a fallback/primary channel when email delivery is unavailable.
type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	var transport http.RoundTripper

	// InsecureSkipVerify permitido APENAS em desenvolvimento explícito (SEC-008).
	// Em produção o client padrão do Go valida TLS normalmente.
	if os.Getenv("ENV") == "development" && os.Getenv("TLS_SKIP_VERIFY") == "true" {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
		log.Println("[WARN] TLS_SKIP_VERIFY ativo — usar apenas em desenvolvimento")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	if transport != nil {
		client.Transport = transport
	}

	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   client,
	}
}

// IsConfigured returns true if both token and chat ID are set.
func (t *TelegramNotifier) IsConfigured() bool {
	return t.botToken != "" && t.chatID != ""
}

// SendOTP sends the OTP code to the specified Telegram chat.
// chatID: se não-vazio, usa este chat_id; caso contrário, usa o chat_id padrão configurado.
func (t *TelegramNotifier) SendOTP(chatID, toEmail, code string) error {
	if t.botToken == "" {
		return nil
	}

	targetChatID := chatID
	if targetChatID == "" {
		targetChatID = t.chatID
	}
	if targetChatID == "" {
		return nil
	}

	text := fmt.Sprintf(
		"🔐 <b>Código de acesso</b>\n\n"+
			"Usuário: <code>%s</code>\n"+
			"Código: <b>%s</b>\n\n"+
			"⏱ Válido por 10 minutos.",
		toEmail, code,
	)

	payload := map[string]interface{}{
		"chat_id":    targetChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: API returned status %d", resp.StatusCode)
	}

	log.Printf("[TELEGRAM] OTP delivered to chat %s for %s", targetChatID, toEmail)
	return nil
}
