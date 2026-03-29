package consumer

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/institutoitinerante/notification-service/internal/model"
	"github.com/institutoitinerante/notification-service/internal/service"
)

// brazilianDateTime formats a RFC3339 string to "02/01/2006 às 15:04" (BRT).
func brazilianDateTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	brt := time.FixedZone("BRT", -3*3600)
	return t.In(brt).Format("02/01/2006 às 15:04")
}

// brazilianDate formats a RFC3339 string to "02/01/2006" (BRT).
func brazilianDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	brt := time.FixedZone("BRT", -3*3600)
	return t.In(brt).Format("02/01/2006")
}

// NotificationHandlers implements all consumer handler interfaces using existing services.
type NotificationHandlers struct {
	otpSvc          *service.OTPService
	notificationSvc *service.NotificationService
}

func NewNotificationHandlers(
	otpSvc *service.OTPService,
	notificationSvc *service.NotificationService,
) *NotificationHandlers {
	return &NotificationHandlers{
		otpSvc:          otpSvc,
		notificationSvc: notificationSvc,
	}
}

// Ensure NotificationHandlers satisfies all handler interfaces.
var _ OTPHandler = (*NotificationHandlers)(nil)
var _ CustomerHandler = (*NotificationHandlers)(nil)
var _ CheckoutHandler = (*NotificationHandlers)(nil)
var _ SchedulingHandler = (*NotificationHandlers)(nil)

// HandleOTPRequested processes otp.requested from auth.events.
func (h *NotificationHandlers) HandleOTPRequested(ctx context.Context, event OTPRequestedEvent) error {
	if _, err := mail.ParseAddress(event.Email); err != nil {
		return fmt.Errorf("invalid email %q in OTPRequestedEvent — discarding: %w", event.Email, err)
	}
	log.Printf("[consumer] otp.requested for %s (service: %s, channel: %s)", event.Email, event.ServiceName, event.Channel)

	expiresAt, err := h.otpSvc.GenerateAndSendChannel(ctx, event.Email, event.Phone, event.Channel, event.TelegramChatID)
	if err != nil {
		return fmt.Errorf("GenerateAndSendChannel OTP for %s (channel: %s): %w", event.Email, event.Channel, err)
	}

	log.Printf("[consumer] OTP sent to %s via %s, expires at %s", event.Email, event.Channel, expiresAt.Format(time.RFC3339))
	return nil
}

// HandleCustomerCreated processes customer.created from customer.events.
func (h *NotificationHandlers) HandleCustomerCreated(ctx context.Context, event CustomerCreatedEvent) error {
	if _, err := mail.ParseAddress(event.Email); err != nil {
		return fmt.Errorf("invalid email %q in CustomerCreatedEvent — discarding: %w", event.Email, err)
	}
	log.Printf("[consumer] customer.created for %s (id: %s, service: %s)", event.Email, event.CustomerID, event.ServiceName)

	name := event.Name
	if name == "" {
		name = event.Email
	}

	const tmplStr = `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#333">Bem-vindo ao {{.Service}}! 🎉</h2>
  <p>Olá, <strong>{{.Name}}</strong>!</p>
  <p>Sua conta foi criada com sucesso. Você já pode fazer login e começar a usar o <strong>{{.Service}}</strong>.</p>
  <p style="color:#999;font-size:12px;margin-top:32px">Instituto Itinerante de Tecnologia — institutoitinerante.com.br</p>
</div>`

	html, err := renderHTML(tmplStr, map[string]string{
		"Name":    name,
		"Service": event.ServiceName,
	})
	if err != nil {
		return fmt.Errorf("render welcome email: %w", err)
	}

	_, err = h.notificationSvc.Send(ctx, &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: event.Email,
		Subject:   fmt.Sprintf("Bem-vindo ao %s!", template.HTMLEscapeString(event.ServiceName)),
		Body:      html,
	})
	if err != nil {
		return fmt.Errorf("send welcome email to %s: %w", event.Email, err)
	}

	log.Printf("[consumer] welcome email sent to %s", event.Email)
	return nil
}

// HandlePaymentConfirmed processes payment.confirmed from checkout.events.
func (h *NotificationHandlers) HandlePaymentConfirmed(ctx context.Context, event PaymentConfirmedEvent) error {
	if _, err := mail.ParseAddress(event.Email); err != nil {
		return fmt.Errorf("invalid email %q in PaymentConfirmedEvent — discarding: %w", event.Email, err)
	}
	log.Printf("[consumer] payment.confirmed order=%s email=%s amount=%.2f", event.OrderID, event.Email, event.Amount)

	name := event.Name
	if name == "" {
		name = event.Email
	}
	currency := event.Currency
	if currency == "" {
		currency = "BRL"
	}

	const tmplStr = `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#333">Pagamento confirmado ✅</h2>
  <p>Olá, <strong>{{.Name}}</strong>!</p>
  <p>Recebemos o seu pagamento com sucesso.</p>
  <table style="width:100%;border-collapse:collapse;margin:20px 0">
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Pedido</td>
        <td style="padding:8px;border-bottom:1px solid #eee;font-weight:bold">{{.OrderID}}</td></tr>
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Serviço</td>
        <td style="padding:8px;border-bottom:1px solid #eee">{{.Service}}</td></tr>
    <tr><td style="padding:8px;color:#666">Valor</td>
        <td style="padding:8px;font-size:20px;font-weight:bold;color:#16a34a">{{.Currency}} {{.Amount}}</td></tr>
  </table>
  <p style="color:#999;font-size:12px;margin-top:32px">Instituto Itinerante de Tecnologia — institutoitinerante.com.br</p>
</div>`

	html, err := renderHTML(tmplStr, map[string]string{
		"Name":     name,
		"OrderID":  event.OrderID,
		"Service":  event.ServiceName,
		"Currency": currency,
		"Amount":   fmt.Sprintf("%.2f", event.Amount),
	})
	if err != nil {
		return fmt.Errorf("render payment email: %w", err)
	}

	_, err = h.notificationSvc.Send(ctx, &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: event.Email,
		Subject:   fmt.Sprintf("Pagamento confirmado — %s", template.HTMLEscapeString(event.ServiceName)),
		Body:      html,
	})
	if err != nil {
		return fmt.Errorf("send payment receipt to %s: %w", event.Email, err)
	}

	log.Printf("[consumer] payment receipt sent to %s for order %s", event.Email, event.OrderID)
	return nil
}

// HandleSubscriptionActivated processes subscription.activated from checkout.events.
func (h *NotificationHandlers) HandleSubscriptionActivated(ctx context.Context, event SubscriptionActivatedEvent) error {
	if _, err := mail.ParseAddress(event.Email); err != nil {
		return fmt.Errorf("invalid email %q in SubscriptionActivatedEvent — discarding: %w", event.Email, err)
	}
	log.Printf("[consumer] subscription.activated id=%s email=%s service=%s", event.SubscriptionID, event.Email, event.ServiceName)

	name := event.Name
	if name == "" {
		name = event.Email
	}

	const tmplStr = `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#333">Assinatura ativada! 🚀</h2>
  <p>Olá, <strong>{{.Name}}</strong>!</p>
  <p>Sua assinatura do <strong>{{.Service}}</strong> está ativa. Você já tem acesso a todos os recursos do plano <strong>{{.Plan}}</strong>.</p>
  <table style="width:100%;border-collapse:collapse;margin:20px 0">
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Plano</td>
        <td style="padding:8px;border-bottom:1px solid #eee;font-weight:bold">{{.Plan}}</td></tr>
    <tr><td style="padding:8px;color:#666">Válido até</td>
        <td style="padding:8px">{{.ValidUntil}}</td></tr>
  </table>
  <p style="color:#999;font-size:12px;margin-top:32px">Instituto Itinerante de Tecnologia — institutoitinerante.com.br</p>
</div>`

	html, err := renderHTML(tmplStr, map[string]string{
		"Name":       name,
		"Service":    event.ServiceName,
		"Plan":       event.PlanName,
		"ValidUntil": brazilianDate(event.ValidUntil),
	})
	if err != nil {
		return fmt.Errorf("render subscription email: %w", err)
	}

	_, err = h.notificationSvc.Send(ctx, &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: event.Email,
		Subject:   fmt.Sprintf("Sua assinatura do %s está ativa!", template.HTMLEscapeString(event.ServiceName)),
		Body:      html,
	})
	if err != nil {
		return fmt.Errorf("send subscription confirmation to %s: %w", event.Email, err)
	}

	log.Printf("[consumer] subscription confirmation sent to %s", event.Email)
	return nil
}

// HandleAppointmentReminder processes reminder.24h from scheduling.events.
func (h *NotificationHandlers) HandleAppointmentReminder(ctx context.Context, event AppointmentReminderEvent) error {
	if _, err := mail.ParseAddress(event.CustomerEmail); err != nil {
		return fmt.Errorf("invalid email %q in AppointmentReminderEvent — discarding: %w", event.CustomerEmail, err)
	}
	log.Printf("[consumer] reminder.24h id=%s email=%s at=%s", event.AppointmentID, event.CustomerEmail, event.ScheduledAt)

	name := event.CustomerName
	if name == "" {
		name = event.CustomerEmail
	}

	const tmplStr = `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#333">Lembrete de agendamento ⏰</h2>
  <p>Olá, <strong>{{.Name}}</strong>!</p>
  <p>Você tem um agendamento amanhã. Confira os detalhes:</p>
  <table style="width:100%;border-collapse:collapse;margin:20px 0">
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Serviço</td>
        <td style="padding:8px;border-bottom:1px solid #eee;font-weight:bold">{{.Service}}</td></tr>
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Profissional</td>
        <td style="padding:8px;border-bottom:1px solid #eee">{{.Provider}}</td></tr>
    <tr><td style="padding:8px;border-bottom:1px solid #eee;color:#666">Data e hora</td>
        <td style="padding:8px;border-bottom:1px solid #eee">{{.ScheduledAt}}</td></tr>
    {{if .Location}}<tr><td style="padding:8px;color:#666">Local</td>
        <td style="padding:8px">{{.Location}}</td></tr>{{end}}
  </table>
  <p style="color:#999;font-size:12px;margin-top:32px">Instituto Itinerante de Tecnologia — institutoitinerante.com.br</p>
</div>`

	html, err := renderHTML(tmplStr, map[string]string{
		"Name":        name,
		"Service":     event.ServiceName,
		"Provider":    event.ProviderName,
		"ScheduledAt": brazilianDateTime(event.ScheduledAt),
		"Location":    event.Location,
	})
	if err != nil {
		return fmt.Errorf("render reminder email: %w", err)
	}

	_, err = h.notificationSvc.Send(ctx, &model.SendNotificationRequest{
		Type:      model.NotificationTypeEmail,
		Recipient: event.CustomerEmail,
		Subject:   fmt.Sprintf("Lembrete: seu agendamento de %s é amanhã", template.HTMLEscapeString(event.ServiceName)),
		Body:      html,
	})
	if err != nil {
		return fmt.Errorf("send appointment reminder to %s: %w", event.CustomerEmail, err)
	}

	log.Printf("[consumer] appointment reminder sent to %s for appointment %s", event.CustomerEmail, event.AppointmentID)
	return nil
}

// renderHTML executes a html/template with the given string data map.
// All values are automatically HTML-escaped by the template engine.
func renderHTML(tmplStr string, data map[string]string) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
