package service

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
