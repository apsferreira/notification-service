# Analysis QA — notification-service

**Agente:** @qa  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. Testes P0 (Críticos — bloqueia deploy)

### P0-NT-001: Email transacional entregue após evento checkout.confirmed

```go
func TestEmailDeliveryOnCheckoutConfirmed(t *testing.T) {
    // Given
    resendMock := newResendMock()
    consumer := setupConsumer(resendMock)
    
    event := amqp.Delivery{
        Body: mustMarshal(map[string]interface{}{
            "event_id":    "evt-001",
            "customer_id": "cust-001",
            "customer_email": "aluno@test.com",
            "order_total": 150.00,
            "order_id":    "ord-001",
        }),
        RoutingKey: "checkout.confirmed",
    }
    
    // When
    err := consumer.processMessage(context.Background(), event)
    
    // Then
    assert.NoError(t, err)
    assert.Equal(t, 1, resendMock.SentCount())
    assert.Equal(t, "aluno@test.com", resendMock.LastRecipient())
    
    // Verificar log no banco
    notif, _ := repo.FindByEventID("evt-001")
    assert.Equal(t, "sent", notif.Status)
}
```

### P0-NT-002: WhatsApp lembrete entregue após scheduling.class.reminder

```go
func TestWhatsAppReminderOnClassReminder(t *testing.T) {
    metaMock := newMetaMock()
    consumer := setupConsumer(metaMock)
    
    event := amqp.Delivery{
        Body: mustMarshal(map[string]interface{}{
            "event_id":    "evt-002",
            "customer_id": "cust-002",
            "customer_phone": "+5571999999999",
            "class_time": "2026-03-04T19:00:00-03:00",
            "instructor": "Mestre João",
        }),
        RoutingKey: "scheduling.class.reminder",
    }
    
    err := consumer.processMessage(context.Background(), event)
    
    assert.NoError(t, err)
    assert.Equal(t, 1, metaMock.SentCount())
    assert.Equal(t, "class_reminder", metaMock.LastTemplateName())
    assert.Equal(t, "+5571999999999", metaMock.LastRecipient())
}
```

### P0-NT-003: Usuário com opt-out não recebe notificação

```go
func TestOptedOutUserDoesNotReceive(t *testing.T) {
    resendMock := newResendMock()
    prefRepo := newPrefRepo()
    prefRepo.SetOptOut("cust-003", "email")
    
    consumer := setupConsumer(resendMock, prefRepo)
    
    event := amqp.Delivery{
        Body: mustMarshal(map[string]interface{}{
            "event_id":       "evt-003",
            "customer_id":    "cust-003",
            "customer_email": "opted-out@test.com",
        }),
        RoutingKey: "checkout.confirmed",
    }
    
    err := consumer.processMessage(context.Background(), event)
    
    assert.NoError(t, err)             // não é erro, é comportamento esperado
    assert.Equal(t, 0, resendMock.SentCount())
    
    notif, _ := repo.FindByEventID("evt-003")
    assert.Equal(t, "opted_out", notif.Status)
}
```

### P0-NT-004: Falha no provedor resulta em NACK (não ACK)

```go
func TestProviderFailureResultsInNACK(t *testing.T) {
    resendMock := newResendMock()
    resendMock.ForceError(errors.New("connection refused"))
    
    delivery := mockDelivery(t, "checkout.confirmed", "evt-004")
    
    consumer := setupConsumer(resendMock)
    consumer.processWithRetry(context.Background(), delivery)
    
    assert.True(t, delivery.NACKed)
    assert.False(t, delivery.ACKed)
}
```

---

## 2. Testes de Deduplicação

### DEDUP-001: Mesmo evento processado duas vezes — segundo é ignorado

```go
func TestDeduplicationPreventsDoubleDelivery(t *testing.T) {
    redisClient := setupTestRedis()
    dedupSvc := NewDeduplicationService(redisClient)
    resendMock := newResendMock()
    consumer := setupConsumer(resendMock, dedupSvc)
    
    event := buildEvent("evt-dedup-001", "cust-001", "checkout.confirmed")
    
    // Primeiro processamento
    err1 := consumer.processMessage(context.Background(), event)
    assert.NoError(t, err1)
    assert.Equal(t, 1, resendMock.SentCount())
    
    // Segundo processamento (redelivery simulado)
    err2 := consumer.processMessage(context.Background(), event)
    assert.NoError(t, err2) // não é erro
    assert.Equal(t, 1, resendMock.SentCount()) // ainda 1, não 2
}
```

### DEDUP-002: Expiração do TTL permite reenvio após 24h

```go
func TestDeduplicationExpiresAfterTTL(t *testing.T) {
    // Usar Redis com clock mock
    fakeClock := clock.NewMock()
    dedupSvc := NewDeduplicationServiceWithClock(redisClient, 24*time.Hour, fakeClock)
    
    event := buildEvent("evt-ttl-001", "cust-001", "checkout.confirmed")
    
    // Primeiro envio
    isDup1, _ := dedupSvc.IsAlreadySent(ctx, "cust-001", "checkout.confirmed", "evt-ttl-001")
    assert.False(t, isDup1)
    
    // Avançar 25 horas
    fakeClock.Add(25 * time.Hour)
    
    // Novo envio deve ser permitido (TTL expirou)
    isDup2, _ := dedupSvc.IsAlreadySent(ctx, "cust-001", "checkout.confirmed", "evt-ttl-001")
    assert.False(t, isDup2) // não é duplicata após expiração
}
```

### DEDUP-003: Eventos com mesmo tipo mas IDs diferentes são enviados

```go
func TestDifferentEventIDsAreNotDeduplicated(t *testing.T) {
    // Dois eventos de checkout para o mesmo cliente com IDs diferentes
    event1 := buildEvent("evt-A", "cust-001", "checkout.confirmed")
    event2 := buildEvent("evt-B", "cust-001", "checkout.confirmed")
    
    consumer.processMessage(ctx, event1)
    consumer.processMessage(ctx, event2)
    
    assert.Equal(t, 2, resendMock.SentCount()) // ambos devem ser enviados
}
```

---

## 3. Testes de Retry

### RETRY-001: Retry com backoff após 5xx do provedor

```go
func TestRetryWithBackoffOn5xx(t *testing.T) {
    resendMock := newResendMock()
    resendMock.ReturnErrorsForN(2, &HTTPError{StatusCode: 500})
    resendMock.ThenSucceed()
    
    fakeClock := clock.NewMock()
    consumer := setupConsumerWithClock(resendMock, fakeClock)
    
    start := time.Now()
    delivery := mockDelivery(t, "checkout.confirmed", "evt-retry-001")
    
    err := consumer.processWithRetry(context.Background(), delivery)
    
    assert.NoError(t, err)
    assert.Equal(t, 3, resendMock.CallCount())         // 2 falhas + 1 sucesso
    assert.True(t, delivery.ACKed)
    
    // Verificar delays: 1s após 1a tentativa, 4s após 2a
    assert.GreaterOrEqual(t, fakeClock.Elapsed(), 5*time.Second)
}
```

### RETRY-002: Máximo de 3 tentativas → DLQ

```go
func TestMaxRetriesExhaustedGoesToDLQ(t *testing.T) {
    resendMock := newResendMock()
    resendMock.AlwaysError(&HTTPError{StatusCode: 503})
    
    delivery := mockDelivery(t, "checkout.confirmed", "evt-retry-002")
    
    err := consumer.processWithRetry(context.Background(), delivery)
    
    assert.Error(t, err)
    assert.Equal(t, 3, resendMock.CallCount())
    assert.True(t, delivery.NACKed)   // vai para DLQ (nack sem requeue)
    assert.False(t, delivery.ACKed)
}
```

### RETRY-003: Erro não-retryável (400) não gera retry

```go
func TestNon5xxErrorIsNotRetried(t *testing.T) {
    resendMock := newResendMock()
    resendMock.AlwaysError(&HTTPError{StatusCode: 400, Body: "invalid email"})
    
    delivery := mockDelivery(t, "checkout.confirmed", "evt-retry-003")
    
    err := consumer.processWithRetry(context.Background(), delivery)
    
    assert.Error(t, err)
    assert.Equal(t, 1, resendMock.CallCount()) // apenas 1 tentativa
    assert.True(t, delivery.NACKed)
}
```

---

## 4. Cobertura Mínima Exigida

| Pacote | Meta |
|--------|------|
| `internal/dedup` | 100% |
| `internal/dispatcher` | 80% |
| `internal/channel/email` | 70% |
| `internal/channel/whatsapp` | 70% |
| `internal/template` | 80% |
| **Total** | **≥70%** |

---

## 5. Testes de Integração (Staging)

Antes do deploy em produção, validar manualmente:

1. Publicar evento `checkout.confirmed` no RabbitMQ → verificar email recebido
2. Publicar evento `scheduling.class.reminder` → verificar WhatsApp recebido
3. Publicar mesmo evento duas vezes → verificar que só 1 mensagem chegou
4. Simular falha do Resend (bloquear no firewall) → verificar retry + DLQ após 3 tentativas
5. Responder "PARAR" no WhatsApp → verificar opt-out registrado e envios cessados
