package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WhatsAppNotifier delivers messages via WhatsApp Business API.
type WhatsAppNotifier interface {
	IsConfigured() bool
	SendOTP(toPhone, code string) error
	SendText(toPhone, message string) error
}

// NoopWhatsAppNotifier é a implementação no-op usada quando o WhatsApp não está configurado.
type NoopWhatsAppNotifier struct{}

func (n *NoopWhatsAppNotifier) IsConfigured() bool         { return false }
func (n *NoopWhatsAppNotifier) SendOTP(_, _ string) error  { return nil }
func (n *NoopWhatsAppNotifier) SendText(_, _ string) error { return nil }

// MetaWhatsAppNotifier integra com a WhatsApp Business Cloud API (Meta Graph API).
// Docs: https://developers.facebook.com/docs/whatsapp/cloud-api/messages/text-messages
// Requer: WHATSAPP_PHONE_NUMBER_ID e WHATSAPP_ACCESS_TOKEN nas variáveis de ambiente.
type MetaWhatsAppNotifier struct {
	phoneNumberID string // ex: "123456789012345"
	accessToken   string // Bearer token do System User
	client        *http.Client
}

// NewMetaWhatsAppNotifier cria um MetaWhatsAppNotifier ou retorna NoopWhatsAppNotifier
// se as credenciais não estiverem presentes.
func NewMetaWhatsAppNotifier(phoneNumberID, accessToken string) WhatsAppNotifier {
	if phoneNumberID == "" || accessToken == "" {
		return &NoopWhatsAppNotifier{}
	}
	return &MetaWhatsAppNotifier{
		phoneNumberID: phoneNumberID,
		accessToken:   accessToken,
		client:        &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *MetaWhatsAppNotifier) IsConfigured() bool { return true }

// SendOTP envia um código OTP via WhatsApp.
func (m *MetaWhatsAppNotifier) SendOTP(toPhone, code string) error {
	msg := fmt.Sprintf("*Código de acesso*\n\n*%s*\n\nVálido por 10 minutos.\n\n_Instituto Itinerante de Tecnologia_", code)
	return m.send(toPhone, msg)
}

// SendText envia uma mensagem de texto livre via WhatsApp.
func (m *MetaWhatsAppNotifier) SendText(toPhone, message string) error {
	return m.send(toPhone, message)
}

func (m *MetaWhatsAppNotifier) send(toPhone, text string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v20.0/%s/messages", m.phoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                toPhone,
		"type":              "text",
		"text":              map[string]string{"body": text},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("whatsapp: falha ao serializar payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp: falha ao criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.accessToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: falha ao enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp: Meta API retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
