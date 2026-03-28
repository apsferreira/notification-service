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
// Provedores suportados: Evolution API (self-hosted), Z-API, Twilio.
type WhatsAppNotifier interface {
	IsConfigured() bool
	SendOTP(toPhone, code string) error
	SendText(toPhone, message string) error
}

// NoopWhatsAppNotifier é a implementação no-op usada quando o WhatsApp não está configurado.
// Permite que o OTPService compile e rode normalmente sem credenciais WhatsApp.
type NoopWhatsAppNotifier struct{}

func (n *NoopWhatsAppNotifier) IsConfigured() bool        { return false }
func (n *NoopWhatsAppNotifier) SendOTP(_, _ string) error { return nil }
func (n *NoopWhatsAppNotifier) SendText(_, _ string) error { return nil }

// EvolutionWhatsAppNotifier integra com o Evolution API (gateway WhatsApp self-hosted).
// Docs: https://doc.evolution-api.com/
// Provisionar na VLAN 30 antes de usar: EVOLUTION_API_URL, EVOLUTION_API_KEY, EVOLUTION_INSTANCE_ID.
type EvolutionWhatsAppNotifier struct {
	baseURL    string // ex: http://192.168.30.xxx:8080
	apiKey     string
	instanceID string
	client     *http.Client
}

// NewEvolutionWhatsAppNotifier cria um EvolutionWhatsAppNotifier ou retorna NoopWhatsAppNotifier
// se as credenciais obrigatórias (baseURL e apiKey) não estiverem presentes.
func NewEvolutionWhatsAppNotifier(baseURL, apiKey, instanceID string) WhatsAppNotifier {
	if baseURL == "" || apiKey == "" {
		return &NoopWhatsAppNotifier{}
	}
	return &EvolutionWhatsAppNotifier{
		baseURL:    baseURL,
		apiKey:     apiKey,
		instanceID: instanceID,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// IsConfigured sempre retorna true quando a struct está inicializada com credenciais.
func (e *EvolutionWhatsAppNotifier) IsConfigured() bool { return true }

// SendOTP envia um código OTP via WhatsApp para o número informado.
// Requer Evolution API provisionada na VLAN 30.
// POST {baseURL}/message/sendText/{instanceID}
func (e *EvolutionWhatsAppNotifier) SendOTP(toPhone, code string) error {
	message := fmt.Sprintf("*Seu codigo de acesso*\n\nCodigo: *%s*\nValido por 10 minutos.\n\n_Instituto Itinerante de Tecnologia_", code)
	return e.sendTextMessage(toPhone, message)
}

// SendText envia uma mensagem de texto livre via WhatsApp.
func (e *EvolutionWhatsAppNotifier) SendText(toPhone, message string) error {
	return e.sendTextMessage(toPhone, message)
}

func (e *EvolutionWhatsAppNotifier) sendTextMessage(toPhone, message string) error {
	url := fmt.Sprintf("%s/message/sendText/%s", e.baseURL, e.instanceID)

	payload := map[string]interface{}{
		"number": toPhone,
		"text":   message,
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
	req.Header.Set("apikey", e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: falha ao enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp: Evolution API retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
