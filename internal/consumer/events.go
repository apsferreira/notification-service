package consumer

// OTPRequestedEvent is published by auth-service when a user requests an OTP code.
// Exchange: auth.events | Routing key: otp.requested
type OTPRequestedEvent struct {
	Email          string `json:"email"`
	Phone          string `json:"phone,omitempty"`           // E.164, obrigatório quando channel == "whatsapp"
	ServiceName    string `json:"service_name"`              // human-readable, e.g. "Instituto Itinerante"
	Channel        string `json:"channel"`                   // "email" | "telegram" | "whatsapp"
	TelegramChatID string `json:"telegram_chat_id,omitempty"` // chat_id do usuário — usado quando channel == "telegram"
}

// CustomerCreatedEvent is published by customer-service when a new customer registers.
// Exchange: customer.events | Routing key: customer.created
type CustomerCreatedEvent struct {
	CustomerID  string `json:"customer_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	ServiceName string `json:"service_name"` // product the customer registered in
}

// PaymentConfirmedEvent is published by checkout-service when a payment is confirmed.
// Exchange: checkout.events | Routing key: payment.confirmed
type PaymentConfirmedEvent struct {
	OrderID     string  `json:"order_id"`
	CustomerID  string  `json:"customer_id"`
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"` // e.g. "BRL"
	ServiceName string  `json:"service_name"`
}

// SubscriptionActivatedEvent is published by checkout-service when a subscription is activated.
// Exchange: checkout.events | Routing key: subscription.activated
type SubscriptionActivatedEvent struct {
	SubscriptionID string `json:"subscription_id"`
	CustomerID     string `json:"customer_id"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	ServiceName    string `json:"service_name"`
	PlanName       string `json:"plan_name"`
	ValidUntil     string `json:"valid_until"` // RFC3339
}

// AppointmentReminderEvent is published by scheduling-service 24h before an appointment.
// Exchange: scheduling.events | Routing key: reminder.24h
type AppointmentReminderEvent struct {
	AppointmentID string `json:"appointment_id"`
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
	ServiceName   string `json:"service_name"`
	ProviderName  string `json:"provider_name"`
	ScheduledAt   string `json:"scheduled_at"` // RFC3339
	Location      string `json:"location,omitempty"`
}
