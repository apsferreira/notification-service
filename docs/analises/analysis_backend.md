# Analysis Backend — notification-service

**Agente:** @backend  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. Arquitetura Geral

```
┌─────────────────────────────────────────────────────┐
│              notification-service                    │
│  ┌──────────────┐   ┌──────────────┐                │
│  │  RabbitMQ    │   │  HTTP API    │                │
│  │  Consumer    │   │  (Fiber :3012)│               │
│  └──────┬───────┘   └──────────────┘                │
│         │                                            │
│  ┌──────▼───────────────────────────────────┐       │
│  │         Notification Dispatcher           │       │
│  │  ┌─────────────┐  ┌──────────────────┐  │       │
│  │  │ Dedup Check │  │ Template Resolver │  │       │
│  │  │  (Redis)    │  │  (PostgreSQL)     │  │       │
│  │  └─────┬───────┘  └────────┬─────────┘  │       │
│  └────────┼───────────────────┼────────────┘       │
│           │                   │                      │
│  ┌────────▼───────────────────▼────────────┐        │
│  │           Channel Adapters               │        │
│  │  ┌──────────┐ ┌───────────┐ ┌────────┐ │        │
│  │  │  Resend  │ │  Meta WA  │ │  Push  │ │        │
│  │  │  (Email) │ │  Cloud    │ │  Expo  │ │        │
│  │  └──────────┘ └───────────┘ └────────┘ │        │
│  └─────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────┘
```

---

## 2. Consumer RabbitMQ

### Configuração de Filas

```go
// Exchanges e filas a consumir
const (
    ExchangeCheckout    = "checkout"
    ExchangeScheduling  = "scheduling"
    
    QueueNotifications  = "notification-service.events"
    
    // Dead Letter
    QueueDLQ           = "notification-service.dlq"
)

// Binding de routing keys
var bindings = []string{
    "checkout.confirmed",
    "checkout.payment.overdue",
    "scheduling.class.reminder",
    "scheduling.checkin.confirmed",
    "trial.expiring",
}
```

### Consumer Loop com Retry

```go
func (c *Consumer) Start(ctx context.Context) error {
    msgs, err := c.ch.Consume(QueueNotifications, "", false, false, false, false, nil)
    if err != nil {
        return err
    }
    
    for {
        select {
        case msg := <-msgs:
            if err := c.processWithRetry(ctx, msg); err != nil {
                log.Error("failed after retries", "error", err)
                msg.Nack(false, false) // → DLQ
            } else {
                msg.Ack(false)
            }
        case <-ctx.Done():
            return nil
        }
    }
}
```

### Configuração da Fila com DLQ

```go
args := amqp.Table{
    "x-dead-letter-exchange":    "notification-service.dlx",
    "x-dead-letter-routing-key": "dead",
    "x-message-ttl":             86400000, // 24h TTL na DLQ
}
```

---

## 3. Retry com Backoff Exponencial (REQ-NT-04)

### Estratégia

- Máximo 3 tentativas antes de mover para DLQ
- Backoff: 1s → 4s → 16s (base 4, expoente = tentativa-1)
- Retry apenas para erros transitórios (5xx, timeout, rate limit)
- Não retentar para erros permanentes (400 Bad Request, opt-out)

```go
type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    Multiplier  float64
}

var DefaultRetry = RetryConfig{
    MaxAttempts: 3,
    BaseDelay:   1 * time.Second,
    Multiplier:  4.0,
}

func (c *Consumer) processWithRetry(ctx context.Context, msg amqp.Delivery) error {
    var lastErr error
    for attempt := 1; attempt <= DefaultRetry.MaxAttempts; attempt++ {
        if err := c.dispatcher.Dispatch(ctx, msg); err != nil {
            if !isRetryable(err) {
                return err // falha permanente, vai para DLQ imediatamente
            }
            lastErr = err
            delay := time.Duration(float64(DefaultRetry.BaseDelay) * 
                math.Pow(DefaultRetry.Multiplier, float64(attempt-1)))
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }
            continue
        }
        return nil
    }
    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error) bool {
    var httpErr *HTTPError
    if errors.As(err, &httpErr) {
        return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
    }
    return errors.Is(err, context.DeadlineExceeded) ||
           errors.Is(err, io.ErrUnexpectedEOF)
}
```

---

## 4. Deduplicação com Redis (REQ-NT-02)

### Estratégia

Chave composta: `notif:{customer_id}:{template_type}:{event_id}`  
TTL: 24h (suficiente para evitar duplicatas de redelivery RabbitMQ)

```go
type DeduplicationService struct {
    redis *redis.Client
    ttl   time.Duration
}

func NewDeduplicationService(r *redis.Client) *DeduplicationService {
    return &DeduplicationService{redis: r, ttl: 24 * time.Hour}
}

func (d *DeduplicationService) IsAlreadySent(ctx context.Context, 
    customerID, templateType, eventID string) (bool, error) {
    
    key := fmt.Sprintf("notif:%s:%s:%s", customerID, templateType, eventID)
    
    // SET NX com TTL — atômico
    set, err := d.redis.SetNX(ctx, key, "1", d.ttl).Result()
    if err != nil {
        return false, err
    }
    return !set, nil // set=false significa que chave já existia → duplicata
}
```

### Redis Database

Usar Redis DB7 (conforme PRD original) para isolamento de namespace.

---

## 5. Schema Completo de Tabelas

```sql
-- Templates de notificação (REQ-NT-01)
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'whatsapp', 'push')),
    subject VARCHAR(200),
    body_template TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    active BOOLEAN DEFAULT true,
    meta_template_id VARCHAR(100), -- ID do template aprovado pela Meta
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(event_type, channel)
);

-- Log de notificações enviadas (REQ-NT-03)
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    template_type VARCHAR(100) NOT NULL,
    event_id VARCHAR(200),          -- ID do evento RabbitMQ para deduplicação
    payload JSONB,                   -- variáveis usadas (sem PII sensível)
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'opted_out')),
    provider_id VARCHAR(200),        -- ID retornado pelo Resend/Meta
    error_message TEXT,
    attempt_count INTEGER DEFAULT 1,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_customer ON notifications(customer_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created ON notifications(created_at);

-- Preferências de notificação (opt-out LGPD)
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    event_category VARCHAR(50) DEFAULT 'transactional',
    opted_in BOOLEAN DEFAULT true,
    opted_out_at TIMESTAMPTZ,
    opt_out_reason VARCHAR(200),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(customer_id, channel, event_category)
);
```

---

## 6. Integração Resend

```go
import "github.com/resend/resend-go/v2"

type EmailAdapter struct {
    client *resend.Client
    from   string
}

func (a *EmailAdapter) Send(ctx context.Context, n *Notification) error {
    params := &resend.SendEmailRequest{
        From:    a.from,
        To:      []string{n.RecipientEmail},
        Subject: n.Subject,
        Html:    n.Body,
    }
    
    resp, err := a.client.Emails.SendWithContext(ctx, params)
    if err != nil {
        return fmt.Errorf("resend error: %w", err)
    }
    
    n.ProviderID = resp.Id
    return nil
}
```

---

## 7. Integração Meta Cloud API

```go
type MetaWAAdapter struct {
    httpClient     *http.Client
    phoneNumberID  string
    accessToken    string
    apiVersion     string
}

func (a *MetaWAAdapter) SendTemplate(ctx context.Context, 
    to, templateName, langCode string, 
    components []TemplateComponent) error {
    
    url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages",
        a.apiVersion, a.phoneNumberID)
    
    payload := map[string]interface{}{
        "messaging_product": "whatsapp",
        "to":               to,
        "type":             "template",
        "template": map[string]interface{}{
            "name":       templateName,
            "language":   map[string]string{"code": langCode},
            "components": components,
        },
    }
    
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+a.accessToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := a.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("meta wa error: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
    }
    return nil
}
```

---

## 8. Estrutura de Diretórios

```
notification-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── consumer/          # RabbitMQ consumer
│   ├── dispatcher/        # Event → notification routing
│   ├── dedup/             # Redis deduplication
│   ├── template/          # Template resolver
│   ├── channel/
│   │   ├── email/         # Resend adapter
│   │   ├── whatsapp/      # Meta Cloud API adapter
│   │   └── push/          # Expo Push adapter
│   ├── repository/        # PostgreSQL DAL
│   └── webhook/           # Resend + Meta webhooks
├── migrations/
├── docker-compose.yml
└── Makefile
```
