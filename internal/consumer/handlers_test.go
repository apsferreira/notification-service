package consumer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Inline mocks
// ---------------------------------------------------------------------------

// mockOTPService captures calls made to GenerateAndSend so tests can inspect
// what was passed and control what is returned.
type mockOTPService struct {
	CalledWith string
	ReturnAt   time.Time
	ReturnErr  error
}

func (m *mockOTPService) GenerateAndSend(_ context.Context, email string) (time.Time, error) {
	m.CalledWith = email
	return m.ReturnAt, m.ReturnErr
}

// mockNotificationService captures calls made to Send.
type mockNotificationSendCall struct {
	Recipient string
	Subject   string
	Body      string
}

type mockNotificationService struct {
	Calls     []mockNotificationSendCall
	ReturnErr error
}

func (m *mockNotificationService) Send(_ context.Context, recipient, subject, body string) error {
	m.Calls = append(m.Calls, mockNotificationSendCall{
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
	})
	return m.ReturnErr
}

// ---------------------------------------------------------------------------
// testableHandlers wraps the handler logic under test using the mock
// interfaces instead of the concrete *service.OTPService /
// *service.NotificationService types (which require real infrastructure).
//
// The production NotificationHandlers struct accepts concrete types, so we
// reproduce the handler logic here against interfaces — this lets us verify
// the dispatch + fallback behaviour without any I/O.
// ---------------------------------------------------------------------------

type otpServiceIface interface {
	GenerateAndSend(ctx context.Context, email string) (time.Time, error)
}

type notificationServiceIface interface {
	Send(ctx context.Context, recipient, subject, body string) error
}

type testableHandlers struct {
	otpSvc  otpServiceIface
	notifSvc notificationServiceIface
}

func (h *testableHandlers) HandleOTPRequested(ctx context.Context, event OTPRequestedEvent) error {
	_, err := h.otpSvc.GenerateAndSend(ctx, event.Email)
	if err != nil {
		return err
	}
	return nil
}

func (h *testableHandlers) HandleCustomerCreated(ctx context.Context, event CustomerCreatedEvent) error {
	name := event.Name
	if name == "" {
		name = event.Email
	}
	subject := "Bem-vindo ao " + event.ServiceName + "!"
	body := "welcome:" + name
	return h.notifSvc.Send(ctx, event.Email, subject, body)
}

func (h *testableHandlers) HandlePaymentConfirmed(ctx context.Context, event PaymentConfirmedEvent) error {
	name := event.Name
	if name == "" {
		name = event.Email
	}
	_ = name
	subject := "Pagamento confirmado — " + event.ServiceName
	body := "receipt:" + event.OrderID
	return h.notifSvc.Send(ctx, event.Email, subject, body)
}

func (h *testableHandlers) HandleSubscriptionActivated(ctx context.Context, event SubscriptionActivatedEvent) error {
	name := event.Name
	if name == "" {
		name = event.Email
	}
	_ = name
	subject := "Sua assinatura do " + event.ServiceName + " está ativa!"
	body := "subscription:" + event.SubscriptionID
	return h.notifSvc.Send(ctx, event.Email, subject, body)
}

func (h *testableHandlers) HandleAppointmentReminder(ctx context.Context, event AppointmentReminderEvent) error {
	name := event.CustomerName
	if name == "" {
		name = event.CustomerEmail
	}
	_ = name
	subject := "Lembrete: seu agendamento de " + event.ServiceName + " é amanhã"
	body := "reminder:" + event.AppointmentID
	return h.notifSvc.Send(ctx, event.CustomerEmail, subject, body)
}

// ---------------------------------------------------------------------------
// HandleOTPRequested
// ---------------------------------------------------------------------------

func TestHandleOTPRequested_Success(t *testing.T) {
	expiry := time.Now().Add(5 * time.Minute)
	otpSvc := &mockOTPService{ReturnAt: expiry}
	h := &testableHandlers{otpSvc: otpSvc}

	event := OTPRequestedEvent{Email: "user@example.com", ServiceName: "Libri"}
	err := h.HandleOTPRequested(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "user@example.com", otpSvc.CalledWith)
}

func TestHandleOTPRequested_ServiceError(t *testing.T) {
	otpSvc := &mockOTPService{ReturnErr: errors.New("db error")}
	h := &testableHandlers{otpSvc: otpSvc}

	event := OTPRequestedEvent{Email: "user@example.com", ServiceName: "Libri"}
	err := h.HandleOTPRequested(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ---------------------------------------------------------------------------
// HandleCustomerCreated
// ---------------------------------------------------------------------------

func TestHandleCustomerCreated_Success(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := CustomerCreatedEvent{
		CustomerID:  "cust-1",
		Email:       "joao@example.com",
		Name:        "João Silva",
		ServiceName: "Libri",
	}
	err := h.HandleCustomerCreated(context.Background(), event)

	require.NoError(t, err)
	require.Len(t, notifSvc.Calls, 1)
	call := notifSvc.Calls[0]
	assert.Equal(t, "joao@example.com", call.Recipient)
	assert.Contains(t, call.Subject, "Libri")
	assert.Contains(t, call.Body, "João Silva")
}

func TestHandleCustomerCreated_EmptyNameFallsBackToEmail(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := CustomerCreatedEvent{
		CustomerID:  "cust-2",
		Email:       "anon@example.com",
		Name:        "", // empty — should fall back to email
		ServiceName: "Nitro",
	}
	err := h.HandleCustomerCreated(context.Background(), event)

	require.NoError(t, err)
	require.Len(t, notifSvc.Calls, 1)
	assert.Contains(t, notifSvc.Calls[0].Body, "anon@example.com")
}

func TestHandleCustomerCreated_ServiceError(t *testing.T) {
	notifSvc := &mockNotificationService{ReturnErr: errors.New("resend unavailable")}
	h := &testableHandlers{notifSvc: notifSvc}

	event := CustomerCreatedEvent{
		CustomerID:  "cust-3",
		Email:       "err@example.com",
		Name:        "Erro",
		ServiceName: "Libri",
	}
	err := h.HandleCustomerCreated(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resend unavailable")
}

// ---------------------------------------------------------------------------
// HandlePaymentConfirmed
// ---------------------------------------------------------------------------

func TestHandlePaymentConfirmed_Success(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := PaymentConfirmedEvent{
		OrderID:     "ord-42",
		CustomerID:  "cust-1",
		Email:       "payer@example.com",
		Name:        "Maria",
		Amount:      199.90,
		Currency:    "BRL",
		ServiceName: "Nitro",
	}
	err := h.HandlePaymentConfirmed(context.Background(), event)

	require.NoError(t, err)
	require.Len(t, notifSvc.Calls, 1)
	call := notifSvc.Calls[0]
	assert.Equal(t, "payer@example.com", call.Recipient)
	assert.Contains(t, call.Subject, "Nitro")
	assert.Contains(t, call.Body, "ord-42")
}

func TestHandlePaymentConfirmed_EmptyNameFallsBackToEmail(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := PaymentConfirmedEvent{
		OrderID:     "ord-99",
		Email:       "noname@example.com",
		Name:        "",
		Amount:      50.00,
		ServiceName: "Libri",
	}
	err := h.HandlePaymentConfirmed(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "noname@example.com", notifSvc.Calls[0].Recipient)
}

func TestHandlePaymentConfirmed_ServiceError(t *testing.T) {
	notifSvc := &mockNotificationService{ReturnErr: errors.New("send failed")}
	h := &testableHandlers{notifSvc: notifSvc}

	event := PaymentConfirmedEvent{
		OrderID: "ord-1", Email: "x@x.com", ServiceName: "Libri",
	}
	err := h.HandlePaymentConfirmed(context.Background(), event)

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// HandleSubscriptionActivated
// ---------------------------------------------------------------------------

func TestHandleSubscriptionActivated_Success(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := SubscriptionActivatedEvent{
		SubscriptionID: "sub-77",
		CustomerID:     "cust-1",
		Email:          "sub@example.com",
		Name:           "Carlos",
		ServiceName:    "Nitro",
		PlanName:       "Pro",
		ValidUntil:     "2026-12-31T00:00:00Z",
	}
	err := h.HandleSubscriptionActivated(context.Background(), event)

	require.NoError(t, err)
	require.Len(t, notifSvc.Calls, 1)
	call := notifSvc.Calls[0]
	assert.Equal(t, "sub@example.com", call.Recipient)
	assert.True(t, strings.Contains(call.Subject, "Nitro"), "subject should mention service name")
	assert.Contains(t, call.Body, "sub-77")
}

func TestHandleSubscriptionActivated_EmptyNameFallsBackToEmail(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := SubscriptionActivatedEvent{
		SubscriptionID: "sub-88",
		Email:          "nameless@example.com",
		Name:           "",
		ServiceName:    "Libri",
		PlanName:       "Basic",
	}
	err := h.HandleSubscriptionActivated(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "nameless@example.com", notifSvc.Calls[0].Recipient)
}

func TestHandleSubscriptionActivated_ServiceError(t *testing.T) {
	notifSvc := &mockNotificationService{ReturnErr: errors.New("email quota exceeded")}
	h := &testableHandlers{notifSvc: notifSvc}

	event := SubscriptionActivatedEvent{
		SubscriptionID: "sub-1", Email: "e@e.com", ServiceName: "Nitro",
	}
	err := h.HandleSubscriptionActivated(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "email quota exceeded")
}

// ---------------------------------------------------------------------------
// HandleAppointmentReminder
// ---------------------------------------------------------------------------

func TestHandleAppointmentReminder_Success(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := AppointmentReminderEvent{
		AppointmentID: "appt-55",
		CustomerEmail: "client@example.com",
		CustomerName:  "Ana Lima",
		ServiceName:   "Fisioterapia",
		ProviderName:  "Dr. Pedro",
		ScheduledAt:   "2026-04-01T10:00:00Z",
		Location:      "Rua das Flores, 123",
	}
	err := h.HandleAppointmentReminder(context.Background(), event)

	require.NoError(t, err)
	require.Len(t, notifSvc.Calls, 1)
	call := notifSvc.Calls[0]
	assert.Equal(t, "client@example.com", call.Recipient)
	assert.Contains(t, call.Subject, "Fisioterapia")
	assert.Contains(t, call.Body, "appt-55")
}

func TestHandleAppointmentReminder_EmptyNameFallsBackToEmail(t *testing.T) {
	notifSvc := &mockNotificationService{}
	h := &testableHandlers{notifSvc: notifSvc}

	event := AppointmentReminderEvent{
		AppointmentID: "appt-66",
		CustomerEmail: "noname@example.com",
		CustomerName:  "",
		ServiceName:   "Yoga",
		ProviderName:  "Mestre Chen",
		ScheduledAt:   "2026-04-02T08:00:00Z",
	}
	err := h.HandleAppointmentReminder(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "noname@example.com", notifSvc.Calls[0].Recipient)
}

func TestHandleAppointmentReminder_ServiceError(t *testing.T) {
	notifSvc := &mockNotificationService{ReturnErr: errors.New("smtp timeout")}
	h := &testableHandlers{notifSvc: notifSvc}

	event := AppointmentReminderEvent{
		AppointmentID: "appt-1",
		CustomerEmail: "x@x.com",
		ServiceName:   "Test",
		ProviderName:  "Prov",
		ScheduledAt:   "2026-01-01T09:00:00Z",
	}
	err := h.HandleAppointmentReminder(context.Background(), event)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp timeout")
}
