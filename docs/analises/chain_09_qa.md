# chain_09_qa.md — @qa
# Plano de Qualidade: notification-service

**Agente:** @qa  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_05_backend.md, chain_08_ai.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam a estratégia de testes:
>
> - **@data:** "Testar dedup: enviar mesmo event_id duas vezes → apenas 1 notificação" → caso de teste explícito
> - **@data:** "Testar opt-out: cliente opted_out não recebe mesmo com evento válido" → caso de teste explícito
> - **@data:** "Testar latência: 95% dos eventos críticos em < 30s" → teste de performance obrigatório
> - **@data:** "North Star: ≥ 95% eventos críticos entregues com sucesso" → meta de confiabilidade guia testes
> - **@finance:** "Monitorar `attempt_count` — cada retry tem custo" → testar que retry não ocorre sem necessidade
> - **@finance:** "Fallback: WhatsApp falha → email gratuito" → testar fallback de canal
> - **@ai:** "Fallback obrigatório: se IA falha → usar template base" → testar degradação graceful de IA
> - **@legal:** "Opt-out deve ser processado imediatamente" → testar timing de processamento

---

## 1. Estratégia de Cobertura

**Meta:** 90% de cobertura de código (acima dos 70% do PRD original — justificado pela criticidade do serviço)

**Justificativa para 90%:**
- Serviço transacional (dinheiro, aulas) — falhas têm impacto direto no cliente
- Custo de bug em produção (duplicata WhatsApp) > custo de escrever teste
- @finance quantificou: sem `payment.confirmed` → R$2.500/mês em suporte

```
Cobertura por camada:
├── Handlers (API)         → 90%
├── Consumers (RabbitMQ)   → 95%
├── Dispatch Engine        → 95%
├── Dedup                  → 100% (crítico)
├── Providers (email/wa/push) → 80% (mocks)
├── Repository             → 85%
└── AI (personalizer)      → 75% (mais difícil de testar)
```

---

## 2. Casos de Teste Críticos

### 2.1 Deduplicação (CRÍTICO — 100% cobertura)

```go
// internal/dedup/dedup_test.go
func TestDedup_FirstMessageAllowed(t *testing.T) {
    // Dado: evento nunca visto antes
    // Quando: verificar se é duplicado
    // Então: retorna isDup=false (permitido)
    isDup, err := dedupSvc.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-1")
    assert.NoError(t, err)
    assert.False(t, isDup)
}

func TestDedup_SameEventBlocked(t *testing.T) {
    // Dado: mesmo event_id enviado novamente
    // Quando: verificar segunda vez
    // Então: retorna isDup=true (bloqueado)
    dedupSvc.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-1") // primeira vez
    isDup, _ := dedupSvc.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-1") // segunda vez
    assert.True(t, isDup)
}

func TestDedup_DifferentEventAllowed(t *testing.T) {
    // Dado: mesmo cliente, mesmo template, event_id DIFERENTE
    // Então: permitido (segundo pagamento do mesmo cliente)
    isDup, _ := dedupSvc.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-2")
    assert.False(t, isDup)
}

func TestDedup_TTLExpiry(t *testing.T) {
    // Dado: Redis com TTL muito curto (para teste)
    // Quando: enviar, esperar TTL, enviar novamente
    // Então: segundo envio é permitido (TTL expirou)
    shortTTLDedup := NewDedupService(redis, 1*time.Millisecond)
    shortTTLDedup.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-3")
    time.Sleep(5 * time.Millisecond)
    isDup, _ := shortTTLDedup.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-3")
    assert.False(t, isDup)
}

func TestDedup_RedisFailure_AllowsMessage(t *testing.T) {
    // Dado: Redis indisponível
    // Quando: verificar dedup
    // Então: permitir (fail-open — melhor duplicata que perder notificação)
    brokenRedis := setupBrokenRedis()
    dedupSvc := NewDedupService(brokenRedis, 24*time.Hour)
    isDup, err := dedupSvc.IsAlreadySent(ctx, "cust-1", "payment.confirmed", "event-4")
    assert.Error(t, err) // erro é retornado
    assert.False(t, isDup) // mas permite envio (fail-open)
}
```

### 2.2 Opt-Out Imediato (CRÍTICO — @legal)

```go
// internal/dispatch/dispatcher_test.go
func TestDispatch_OptedOutCustomerNotNotified(t *testing.T) {
    // Dado: cliente opted_out de WhatsApp
    prefs := &Preferences{WhatsApp: false, Email: true}
    mockPrefsRepo.On("GetPreferences", "cust-opted-out").Return(prefs, nil)
    
    // Quando: evento class.reminder (primário WhatsApp)
    err := dispatcher.Dispatch(ctx, "cust-opted-out", "class.reminder", data)
    
    // Então: não envia WhatsApp, pode enviar email
    assert.NoError(t, err)
    mockWhatsApp.AssertNotCalled(t, "Send") // WhatsApp NÃO chamado
}

func TestDispatch_OptedOutAllChannels_NoNotificationSent(t *testing.T) {
    // Dado: cliente opted_out de TODOS os canais
    prefs := &Preferences{WhatsApp: false, Email: false, Push: false}
    
    // Quando: qualquer evento
    // Então: nenhum provider é chamado
    dispatcher.Dispatch(ctx, "cust-all-optout", "trial.expiring", data)
    
    mockWhatsApp.AssertNotCalled(t, "Send")
    mockEmail.AssertNotCalled(t, "Send")
    mockPush.AssertNotCalled(t, "Send")
}

func TestOptOut_ProcessedImmediately(t *testing.T) {
    // Dado: cliente recebe notificação e logo depois faz opt-out
    // Quando: opt-out via endpoint
    // Então: próxima notificação NÃO é enviada
    
    // 1. Envia notificação (sucesso)
    dispatcher.Dispatch(ctx, "cust-1", "class.reminder", data)
    mockWhatsApp.AssertCalled(t, "Send", mock.Anything)
    
    // 2. Cliente faz opt-out
    optOutHandler.ProcessOptOut(ctx, "cust-1", "whatsapp")
    
    // 3. Tenta enviar novamente
    mockWhatsApp.Calls = nil // reset
    dispatcher.Dispatch(ctx, "cust-1", "class.reminder", data)
    
    // 4. Não enviou
    mockWhatsApp.AssertNotCalled(t, "Send")
}
```

### 2.3 Fallback de Canal (CRÍTICO — @finance)

```go
func TestDispatch_WhatsAppFailsFallsBackToEmail(t *testing.T) {
    // Dado: WhatsApp retorna erro
    mockWhatsApp.On("Send", mock.Anything).Return("", errors.New("evolution api down"))
    mockEmail.On("Send", mock.Anything).Return("email-id-123", nil)
    
    // Quando: class.reminder (canal primário WhatsApp, fallback Email)
    err := dispatcher.Dispatch(ctx, "cust-1", "class.reminder", data)
    
    // Então: sem erro (fallback ok), email enviado
    assert.NoError(t, err)
    mockWhatsApp.AssertCalled(t, "Send")  // tentou WhatsApp
    mockEmail.AssertCalled(t, "Send")     // caiu para email
}

func TestDispatch_AllChannelsFail_ReturnsError(t *testing.T) {
    // Dado: todos os canais falham
    mockWhatsApp.On("Send", mock.Anything).Return("", errors.New("down"))
    mockEmail.On("Send", mock.Anything).Return("", errors.New("down"))
    
    err := dispatcher.Dispatch(ctx, "cust-1", "trial.expiring", data)
    
    assert.Error(t, err)
    // Notificação gravada com status=failed na DB
}
```

### 2.4 Dead-Letter Queue

```go
func TestConsumer_MaxRetriesReached_SendsToDLQ(t *testing.T) {
    // Dado: handler falha 3 vezes (MaxRetries=3)
    failingHandler := func(ctx context.Context, payload []byte) error {
        return errors.New("provider error")
    }
    
    consumer := NewConsumer(config, failingHandler)
    consumer.MaxRetries = 3
    
    // Quando: processar mensagem
    msg := createTestMessage("event-123")
    consumer.processWithRetry(ctx, msg)
    
    // Então: mensagem foi para DLQ, status=failed na DB
    assert.True(t, mockDLQ.WasCalled())
    notif := repo.Get(ctx, "event-123")
    assert.Equal(t, "failed", notif.Status)
    assert.Equal(t, 3, notif.AttemptCount)
}

func TestConsumer_ExponentialBackoff(t *testing.T) {
    // Verificar que os delays são 5s, 30s, 120s
    delays := consumer.getRetryDelays()
    assert.Equal(t, 5*time.Second, delays[0])
    assert.Equal(t, 30*time.Second, delays[1])
    assert.Equal(t, 2*time.Minute, delays[2])
}
```

### 2.5 IA — Degradação Graceful (@ai)

```go
func TestPersonalizer_GeminiFailure_UsesBaseMessage(t *testing.T) {
    // Dado: Gemini API retorna erro
    mockGemini.On("GenerateContent", mock.Anything).Return(nil, errors.New("rate limit"))
    
    // Quando: personalizar mensagem
    result, err := personalizer.PersonalizeMessage(ctx, PersonalizeRequest{
        TemplateType: "trial.expiring",
        BaseMessage:  "Seu trial expira em 2 dias",
    })
    
    // Então: retorna mensagem base SEM erro (não falha a notificação)
    assert.NoError(t, err)
    assert.Equal(t, "Seu trial expira em 2 dias", result)
}

func TestPersonalizer_UrgentTemplate_SkipsAI(t *testing.T) {
    // payment.confirmed nunca chama IA
    personalizer.PersonalizeMessage(ctx, PersonalizeRequest{
        TemplateType: "payment.confirmed",
    })
    mockGemini.AssertNotCalled(t, "GenerateContent")
}
```

---

## 3. Testes de Integração

```go
// tests/integration/notification_flow_test.go
// Requer: PostgreSQL, Redis, RabbitMQ de teste (docker-compose.test.yml)

func TestFullFlow_PaymentConfirmed(t *testing.T) {
    // 1. Publicar evento no RabbitMQ
    publishEvent("checkout.events", "payment.confirmed", PaymentEvent{
        EventID:    "test-event-1",
        CustomerID: testCustomerID,
        Amount:     149.90,
    })
    
    // 2. Aguardar processamento
    time.Sleep(2 * time.Second)
    
    // 3. Verificar que notificação foi criada na DB
    notif := db.QueryRow("SELECT status FROM notifications WHERE customer_id=$1", testCustomerID)
    assert.Equal(t, "sent", notif.Status)
    
    // 4. Verificar que foi enviado via provider mock
    assert.True(t, mockEmailProvider.WasCalled())
    
    // 5. Verificar dedup: publicar mesmo evento novamente
    publishEvent("checkout.events", "payment.confirmed", PaymentEvent{
        EventID: "test-event-1", // mesmo ID!
    })
    time.Sleep(2 * time.Second)
    
    // 6. Verificar que apenas 1 notificação existe
    count := db.QueryRow("SELECT COUNT(*) FROM notifications WHERE customer_id=$1", testCustomerID)
    assert.Equal(t, 1, count)
}
```

---

## 4. Testes de Performance (North Star @data)

```go
// tests/performance/latency_test.go
func TestLatency_95PercentUnder30s(t *testing.T) {
    const numEvents = 100
    latencies := make([]time.Duration, 0, numEvents)
    
    for i := 0; i < numEvents; i++ {
        start := time.Now()
        
        publishEvent("checkout.events", "payment.confirmed", PaymentEvent{
            EventID: fmt.Sprintf("perf-event-%d", i),
        })
        
        // Aguardar notificação aparecer na DB com status=sent
        waitForNotification(t, fmt.Sprintf("perf-event-%d", i), 35*time.Second)
        
        latencies = append(latencies, time.Since(start))
    }
    
    // Calcular p95
    sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
    p95Index := int(float64(numEvents) * 0.95)
    p95 := latencies[p95Index]
    
    assert.Less(t, p95, 30*time.Second, "p95 latência deve ser < 30s (North Star @data)")
}
```

---

## 5. Testes de Regressão (Contrato)

Garantir que mudanças no schema do payload não quebram consumers:

```go
// tests/contract/payload_test.go
func TestPaymentConfirmedPayload_RequiredFields(t *testing.T) {
    // Payload sem event_id deve ser rejeitado (dedup requer)
    invalidPayload := `{"customer_id": "uuid", "amount": 100}` // sem event_id
    err := consumer.Validate([]byte(invalidPayload))
    assert.Error(t, err, "event_id é obrigatório (@data)")
}
```

---

## 6. Organização dos Testes

```
notification-service/
├── internal/
│   ├── dedup/
│   │   └── dedup_test.go        (100% cobertura alvo)
│   ├── dispatch/
│   │   └── dispatcher_test.go   (95% cobertura alvo)
│   ├── consumer/
│   │   └── consumer_test.go     (95% cobertura alvo)
│   └── provider/
│       ├── email/resend_test.go  (80% — mock externo)
│       └── whatsapp/test.go      (80% — mock externo)
├── tests/
│   ├── integration/              (requer infra de teste)
│   ├── performance/              (CI weekly)
│   └── contract/                 (CI em cada PR)
└── mocks/
    ├── email_provider_mock.go
    ├── whatsapp_provider_mock.go
    └── push_provider_mock.go
```

---

## 7. Automação e CI

```yaml
# CI: rodar em cada PR
- Unit tests + coverage check (≥90%)
- Contract tests
- Lint (golangci-lint)

# CI: weekly
- Integration tests (docker-compose.test.yml)
- Performance tests

# Ambiente de teste isolado:
# docker-compose.test.yml com PostgreSQL, Redis, RabbitMQ em memória
```

---

## 8. Definition of Done para notification-service

- [ ] Cobertura ≥ 90% (`go test ./... -cover`)
- [ ] Todos os casos críticos (dedup, opt-out, fallback, DLQ) com testes dedicados
- [ ] Teste de performance: p95 < 30s com 100 eventos
- [ ] Testes de contrato validando payload RabbitMQ
- [ ] Zero erros no golangci-lint
- [ ] Integration tests passando no ambiente de test
