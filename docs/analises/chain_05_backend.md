# chain_05_backend.md — @backend
# Especificação Técnica Backend: notification-service

**Agente:** @backend  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_04_legal.md

---

## Construindo sobre @data e @finance

> Citações diretas que guiam as decisões técnicas:
>
> - **@data:** "Redis DB7 para dedup: chave `notif:{customer_id}:{template_type}:{event_id}` TTL 24h" → implementar exatamente este padrão
> - **@data:** "Campo `event_id` no payload RabbitMQ é obrigatório para dedup correto" → validação no consumer
> - **@data:** "Status `delivered` só atualizado via webhook do provider" → não confundir com `sent`
> - **@data:** "North Star: ≥ 95% eventos críticos entregues em < 30s" → SLA a respeitar na arquitetura
> - **@finance:** "Fallback strategy: WhatsApp falha → tentar email (gratuito)" → implementar fallback por canal
> - **@finance:** "Monitorar `attempt_count` — cada retry tem custo" → campo crítico para controle de gastos
> - **@legal:** "Opt-out deve ser processado imediatamente" → verificar `notification_preferences` antes de enviar
> - **@legal:** "Templates Meta devem ser aprovados antes do uso" → validar template_type antes do envio WhatsApp

---

## 1. Arquitetura Geral

```
┌─────────────────────────────────────────────────────────────────────┐
│                        notification-service                          │
│                                                                     │
│  ┌──────────────┐   ┌───────────────┐   ┌────────────────────────┐ │
│  │  HTTP API    │   │ RabbitMQ      │   │   Dispatch Engine      │ │
│  │  (Fiber)     │   │ Consumers     │   │                        │ │
│  │              │   │               │   │ ┌──────────────────┐   │ │
│  │ POST /notify │   │ payment.*     │──▶│ │ Dedup (Redis)    │   │ │
│  │ GET /notifs  │   │ class.*       │   │ │ OptOut Check     │   │ │
│  │ POST /opt-out│   │ order.*       │   │ │ Template Resolve │   │ │
│  │              │   │ trial.*       │   │ │ Channel Select   │   │ │
│  └──────────────┘   └───────────────┘   │ └──────────────────┘   │ │
│                                          │          │              │ │
│                                          │    ┌─────┴──────┐      │ │
│                                          │    ▼            ▼      │ │
│                                    ┌──────────┐  ┌──────────────┐ │ │
│                                    │  Resend  │  │ Evolution/   │ │ │
│                                    │  (Email) │  │ Meta Cloud   │ │ │
│                                    └──────────┘  └──────────────┘ │ │
│                                          │                         │ │
│                                    ┌──────────┐                    │ │
│                                    │   Expo   │                    │ │
│                                    │  (Push)  │                    │ │
│                                    └──────────┘                    │ │
└─────────────────────────────────────────────────────────────────────┘
         │                    │                    │
    notification_db      Redis DB7          dead_letter_queue
```

---

## 2. Stack e Dependências

```
Runtime:  Go 1.23
Framework: Fiber v2
Database:  PostgreSQL (notification_db)
Cache:     Redis DB7 (dedup)
Queue:     RabbitMQ (amqp091-go)
Email:     Resend SDK (github.com/resendlabs/resend-go)
WhatsApp:  Evolution API REST / Meta Cloud API REST
Push:      Expo Push API REST (HTTP direto)
```

### go.mod principais:
```go
require (
    github.com/gofiber/fiber/v2 v2.52.0
    github.com/rabbitmq/amqp091-go v1.9.0
    github.com/resendlabs/resend-go v1.7.0
    github.com/redis/go-redis/v9 v9.3.0
    github.com/jackc/pgx/v5 v5.5.0
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
)
```

---

## 3. Consumers RabbitMQ

### 3.1 Estrutura do Consumer Base

```go
// internal/consumer/base_consumer.go
type ConsumerConfig struct {
    Exchange    string
    RoutingKey  string
    QueueName   string
    DLQName     string
    MaxRetries  int
    Handler     MessageHandler
}

type MessageHandler func(ctx context.Context, payload []byte) error

func (c *Consumer) Start(ctx context.Context) error {
    msgs, err := c.channel.Consume(c.config.QueueName, "", false, false, false, false, nil)
    for msg := range msgs {
        if err := c.processWithRetry(ctx, msg); err != nil {
            c.sendToDLQ(msg)
        }
    }
}
```

### 3.2 Consumers por Tipo de Evento

```go
// Consumers registrados no main.go
consumers := []ConsumerConfig{
    {
        Exchange:   "checkout.events",
        RoutingKey: "payment.confirmed",
        QueueName:  "notification.payment.confirmed",
        DLQName:    "notification.dlq",
        MaxRetries: 3,
        Handler:    handlers.PaymentConfirmed,
    },
    {
        Exchange:   "scheduling.events",
        RoutingKey: "class.reminder",
        QueueName:  "notification.class.reminder",
        DLQName:    "notification.dlq",
        MaxRetries: 3,
        Handler:    handlers.ClassReminder,
    },
    {
        Exchange:   "order.events",
        RoutingKey: "order.ready",
        QueueName:  "notification.order.ready",
        DLQName:    "notification.dlq",
        MaxRetries: 3,
        Handler:    handlers.OrderReady,
    },
    {
        Exchange:   "checkout.events",
        RoutingKey: "trial.expiring",
        QueueName:  "notification.trial.expiring",
        DLQName:    "notification.dlq",
        MaxRetries: 3,
        Handler:    handlers.TrialExpiring,
    },
}
```

### 3.3 Retry com Backoff Exponencial

```go
// internal/consumer/retry.go
func (c *Consumer) processWithRetry(ctx context.Context, msg amqp091.Delivery) error {
    var attempt int
    if h := msg.Headers["x-retry-count"]; h != nil {
        attempt = int(h.(int64))
    }

    err := c.config.Handler(ctx, msg.Body)
    if err == nil {
        msg.Ack(false)
        return nil
    }

    if attempt >= c.config.MaxRetries {
        msg.Nack(false, false) // vai para DLQ
        return fmt.Errorf("max retries exceeded: %w", err)
    }

    // Backoff: 5s, 30s, 120s
    delays := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}
    delay := delays[min(attempt, len(delays)-1)]
    time.Sleep(delay)

    // Re-publicar com contador incrementado
    c.republishWithRetry(msg, attempt+1)
    msg.Ack(false)
    return nil
}
```

---

## 4. Deduplicação via Redis

```go
// internal/dedup/dedup.go
type DedupService struct {
    redis *redis.Client
    ttl   time.Duration // 24h
}

func (d *DedupService) IsAlreadySent(ctx context.Context, customerID, templateType, eventID string) (bool, error) {
    key := fmt.Sprintf("notif:%s:%s:%s", customerID, templateType, eventID)
    result, err := d.redis.SetNX(ctx, key, "1", d.ttl).Result()
    if err != nil {
        return false, err
    }
    // SetNX retorna true se criou (não existia = não duplicado)
    // retorna false se já existia = duplicado
    return !result, nil
}

// Uso no handler:
func (h *Handler) PaymentConfirmed(ctx context.Context, payload []byte) error {
    var event PaymentConfirmedEvent
    json.Unmarshal(payload, &event)

    isDup, err := h.dedup.IsAlreadySent(ctx, event.CustomerID, "payment.confirmed", event.EventID)
    if isDup {
        log.Info("skipping duplicate notification", "event_id", event.EventID)
        return nil
    }

    return h.dispatch(ctx, event.CustomerID, "payment.confirmed", event)
}
```

---

## 5. Dispatch Engine

```go
// internal/dispatch/dispatcher.go
func (d *Dispatcher) Dispatch(ctx context.Context, customerID, templateType string, data interface{}) error {
    // 1. Verificar opt-out (LGPD - @legal)
    prefs, err := d.repo.GetPreferences(ctx, customerID)
    if err != nil {
        return err
    }

    // 2. Determinar canais por template (ordem de prioridade)
    channels := d.getChannelsForTemplate(templateType, prefs)

    // 3. Criar registro na DB
    notif := &Notification{
        ID:           uuid.New(),
        CustomerID:   customerID,
        TemplateType: templateType,
        Status:       "pending",
    }
    d.repo.Create(ctx, notif)

    // 4. Tentar canais em ordem, fallback se falhar
    for _, ch := range channels {
        err := d.sendViaChannel(ctx, ch, notif, data)
        if err == nil {
            break // sucesso — não precisa fallback
        }
        log.Warn("channel failed, trying fallback", "channel", ch, "error", err)
    }

    return nil
}

// Canais por template com fallback (@finance: WhatsApp falha → email)
func (d *Dispatcher) getChannelsForTemplate(templateType string, prefs *Preferences) []string {
    channelMap := map[string][]string{
        "payment.confirmed":  {"email", "whatsapp"}, // email primeiro (obrigatório)
        "class.reminder":     {"whatsapp", "push", "email"}, // WhatsApp primário
        "order.ready":        {"whatsapp", "push"},
        "trial.expiring":     {"whatsapp", "email"},
        "payment.overdue":    {"whatsapp", "email"},
    }
    all := channelMap[templateType]
    var allowed []string
    for _, ch := range all {
        if prefs.IsOptedIn(ch) {
            allowed = append(allowed, ch)
        }
    }
    return allowed
}
```

---

## 6. Providers

### 6.1 Email — Resend
```go
// internal/provider/email/resend.go
func (p *ResendProvider) Send(ctx context.Context, to, subject, html string) (string, error) {
    params := &resend.SendEmailRequest{
        From:    "IIT <noreply@iit.com.br>",
        To:      []string{to},
        Subject: subject,
        Html:    html,
    }
    resp, err := p.client.Emails.Send(params)
    if err != nil {
        return "", err
    }
    return resp.Id, nil
}
```

### 6.2 WhatsApp — Evolution API
```go
// internal/provider/whatsapp/evolution.go
func (p *EvolutionProvider) SendText(ctx context.Context, phone, message string) (string, error) {
    body := map[string]interface{}{
        "number":  phone,
        "text":    message,
        "delay":   1000,
    }
    resp, err := p.httpClient.Post(
        fmt.Sprintf("%s/message/sendText/%s", p.baseURL, p.instance),
        body,
    )
    // ... parse response
    return resp.Key.ID, nil
}
```

### 6.3 Push — Expo
```go
// internal/provider/push/expo.go
func (p *ExpoProvider) Send(ctx context.Context, token, title, body string) error {
    msg := map[string]interface{}{
        "to":    token,
        "title": title,
        "body":  body,
        "sound": "default",
    }
    _, err := p.httpClient.Post("https://exp.host/--/api/v2/push/send", msg)
    return err
}
```

---

## 7. API HTTP

### 7.1 POST /api/v1/notify (envio direto)
```go
// Usado por serviços internos (M2M com service token)
type NotifyRequest struct {
    CustomerID   string      `json:"customer_id" validate:"required,uuid"`
    TemplateType string      `json:"template_type" validate:"required"`
    Channel      string      `json:"channel"`      // opcional, usa default do template
    Data         interface{} `json:"data"`
}

// POST /api/v1/notify
// Headers: X-Service-Token: <token>
// Response: 202 Accepted { notification_id }
```

### 7.2 GET /api/v1/notifications (histórico)
```
GET /api/v1/notifications?customer_id=uuid&channel=whatsapp&limit=50&offset=0
Authorization: Bearer <jwt>

Response: {
    "notifications": [...],
    "total": 150,
    "page": 1
}
```

### 7.3 POST /api/v1/notifications/opt-out (sem auth — token único)
```go
// URL: /api/v1/notifications/opt-out?token=<one-time-token>&channel=email
// Processa opt-out imediato, atualiza notification_preferences
// Response: 200 + página de confirmação HTML (sem JS)
```

---

## 8. Dead-Letter Queue

```go
// internal/consumer/dlq_processor.go
// Roda como goroutine separada, processa DLQ a cada 1h
func (p *DLQProcessor) Process(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        msgs := p.consumeDLQ(ctx)
        for _, msg := range msgs {
            // Atualizar status na DB como 'failed'
            p.repo.UpdateStatus(ctx, msg.NotificationID, "failed", msg.Error)
            // Alertar via métricas (Prometheus counter)
            p.metrics.DLQProcessed.Inc()
        }
    }
}
```

---

## 9. Variáveis de Ambiente

```env
# Database
DATABASE_URL=postgres://user:pass@shared-postgres:5432/notification_db

# Redis
REDIS_URL=redis://shared-redis:6379/7

# RabbitMQ
RABBITMQ_URL=amqp://user:pass@shared-rabbitmq:5672

# Resend
RESEND_API_KEY=re_xxxx

# Evolution API
EVOLUTION_API_URL=http://shared-evolution-api:8081
EVOLUTION_API_KEY=xxxx
EVOLUTION_INSTANCE=iit-main

# Meta Cloud API (futuro)
META_PHONE_NUMBER_ID=
META_ACCESS_TOKEN=

# Expo Push
EXPO_ACCESS_TOKEN=  # opcional para tier básico

# Service
SERVICE_TOKEN=xxxx  # para autenticação M2M
PORT=3012
```

---

## 10. Estrutura de Pastas

```
notification-service/
├── cmd/
│   └── server/main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   └── middleware/
│   ├── consumer/
│   │   ├── base_consumer.go
│   │   ├── retry.go
│   │   └── dlq_processor.go
│   ├── dispatch/
│   │   └── dispatcher.go
│   ├── dedup/
│   │   └── dedup.go
│   ├── provider/
│   │   ├── email/resend.go
│   │   ├── whatsapp/evolution.go
│   │   ├── whatsapp/meta_cloud.go
│   │   └── push/expo.go
│   ├── repository/
│   │   └── notification_repo.go
│   ├── templates/
│   │   └── *.html (email templates)
│   └── domain/
│       └── notification.go
├── migrations/
├── Dockerfile
└── docker-compose.yml
```
