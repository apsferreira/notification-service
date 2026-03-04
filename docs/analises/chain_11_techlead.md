# chain_11_techlead.md — @techlead
# Decisões Técnicas e Go-Live: notification-service

**Agente:** @techlead  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_05_backend.md, chain_08_ai.md, chain_09_qa.md

---

## Alinhamento com @data e @finance

> Citações diretas que fundamentam os ADRs:
>
> - **@data:** "North Star: ≥ 95% eventos críticos entregues em < 30s" → SLA não negociável define arquitetura
> - **@data:** "WhatsApp open rate 80% vs email 35%" → fundamenta escolha de canal primário
> - **@data:** "Redis DB7 para dedup, chave específica, TTL 24h" → decisão já tomada, implementar exatamente assim
> - **@data:** "Volume M3: ~5.000 notif/mês" → sizing atual não requer escala horizontal ainda
> - **@finance:** "Evolution API até 2.000 conversas/mês, depois migrar Meta Cloud API" → decisão de canal com gatilho claro
> - **@finance:** "Custo de retry: monitorar attempt_count" → retry policy deve equilibrar confiabilidade e custo
> - **@finance:** "TCO (Total Cost of Ownership) deve ser considerado na escolha de canal" → ADR de canal
> - **@qa:** "Cobertura 90%, dedup 100%, testes de contrato em cada PR" → gates de qualidade no CI

---

## 1. ADRs (Architecture Decision Records)

### ADR-001: Canal Primário — WhatsApp via Evolution API (MVP) → Meta Cloud API (escala)

**Contexto:**  
Precisamos de um canal de alta abertura para notificações críticas. WhatsApp tem 80% de open rate (@data). Duas opções: Evolution API (self-hosted, baixo custo) ou Meta Cloud API (oficial, maior custo).

**Decisão:**  
**Fase MVP (M1-M6):** Evolution API  
**Fase Escala (M6+):** Migrar para Meta Cloud API quando volume > 2.000 conversas/mês (@finance)

**Motivo:**
- Evolution API: zero custo adicional no MVP, infra já existente, sem necessidade de Business Verification
- Meta Cloud API: templates pré-aprovados obrigatórios (@legal: 1-7 dias de aprovação = bloqueador de cronograma)
- Com Evolution: go-live imediato. Com Meta: precisa de 10+ dias de antecedência para aprovação

**Consequências:**
- ✅ Go-live mais rápido
- ⚠️ Risco de ban do número com volume alto (mitigar: limite de 500 msg/dia por número)
- ⚠️ Evolution não é API oficial Meta — risco de ToS
- Design do código deve abstrair o provider (interface) para migração fácil

**Interface de abstração obrigatória:**
```go
type WhatsAppProvider interface {
    SendText(ctx context.Context, phone, message string) (string, error)
    SendTemplate(ctx context.Context, phone, templateName string, vars map[string]string) (string, error)
}
// Implementações: EvolutionProvider, MetaCloudProvider
// Swap via env var: WHATSAPP_PROVIDER=evolution|meta_cloud
```

**Gatilho de migração:**  
Quando qualquer condição for atingida:
1. Volume WhatsApp > 2.000 conversas/mês (custo Evolution > Meta Cloud começa a inverter)
2. Primeiro ban de número por Evolution
3. Decisão estratégica de usar templates oficiais para @growth

---

### ADR-002: Retry Policy — 3 tentativas com backoff 5s/30s/120s

**Contexto:**  
Providers externos falham ocasionalmente. Precisamos de retry sem sobrecarregar providers nem gerar duplicatas.

**Decisão:**  
**3 tentativas máximas** com backoff exponencial: 5s → 30s → 120s → DLQ

**Motivo:**
- **@finance:** "Cada retry tem custo" — limitar retries evita custo exponencial
- **@data:** "North Star < 30s" — retries com backoff longo prejudicam a meta, mas são necessários
- Solução: eventos críticos (payment.confirmed, order.ready) têm retry mais rápido (5s/15s/60s)
- Eventos não-críticos (class.reminder 24h antes): podem esperar mais (5s/60s/300s)

**Configuração por tipo de evento:**
```go
var retryConfigs = map[string]RetryConfig{
    "payment.confirmed": {MaxAttempts: 3, Delays: []time.Duration{5*time.Second, 15*time.Second, 60*time.Second}},
    "order.ready":       {MaxAttempts: 3, Delays: []time.Duration{5*time.Second, 15*time.Second, 60*time.Second}},
    "class.reminder":    {MaxAttempts: 3, Delays: []time.Duration{5*time.Second, 30*time.Second, 2*time.Minute}},
    "trial.expiring":    {MaxAttempts: 3, Delays: []time.Duration{5*time.Second, 30*time.Second, 2*time.Minute}},
}
```

**Consequências:**
- ✅ Mensagens críticas têm retry mais rápido
- ✅ Custo controlado (máx 3 chamadas de API por evento)
- ✅ DLQ captura falhas definitivas para análise

---

### ADR-003: Deduplicação via Redis SETNX (fail-open)

**Contexto:**  
RabbitMQ pode redelivery mensagens em cenários de falha. Precisamos evitar notificações duplicadas sem sacrificar entregabilidade.

**Decisão:**  
**Redis SETNX** com chave `notif:{customer_id}:{template_type}:{event_id}` e TTL 24h.  
**Fail-open:** se Redis estiver indisponível → permitir envio (aceitar risco de duplicata vs. perda de notificação)

**Motivo:**
- @data definiu a chave e TTL exatamente → implementar sem desvios
- Fail-open: perder uma notificação importante (payment.confirmed) é pior do que cliente receber duplicata
- Duplicata é recuperável (cliente ignora ou reclama → @support resolve)
- Perda de notificação de pagamento → cliente sem recibo → ticket de suporte + dano à confiança

**Consequências:**
- ✅ Zero notificações perdidas por falha de Redis
- ⚠️ Risco baixo de duplicata se Redis falhar (~0,01% dos casos)
- ✅ TTL de 24h evita acúmulo de chaves no Redis

---

### ADR-004: IA como Enhancement Opcional (não no caminho crítico)

**Contexto:**  
@ai recomendou personalização com Gemini Flash Lite. Precisamos garantir que falha de LLM não afeta entregabilidade.

**Decisão:**  
**IA é bypass-able** em qualquer cenário de falha ou latência. Timeout de 4s para chamadas IA.

**Motivo:**
- @data: "North Star < 30s" — latência de LLM não pode comprometer SLA
- @ai: "Fallback obrigatório" — já previsto no design
- @finance: ROI de personalização é real mas secundário vs. entregabilidade

**Regras:**
1. `payment.confirmed` e `order.ready` → nunca chamam IA (urgentes)
2. Timeout de 4s na chamada Gemini → se exceder, usar template base
3. Feature flag `AI_PERSONALIZATION_ENABLED` → desabilitar em emergência
4. Custo mensal de IA: ~$0,26 com 5k notificações → aprovado por @finance

---

## 2. Decisões de Arquitetura Menores

| Decisão | Escolha | Alternativa Rejeitada | Motivo |
|---------|---------|----------------------|--------|
| Framework HTTP | Fiber v2 | Echo, Gin | Padrão do ecossistema (todos os serviços Go usam Fiber) |
| ORM | pgx/v5 direto | GORM | Performance + controle total sobre queries |
| Consumer | amqp091-go | streadway/amqp | Mantido ativamente, API similar |
| Redis client | go-redis/v9 | redigo | API moderna, suporte a context |
| Template engine | html/template | templ | Suficiente para email; sem overhead |
| Metrics | Prometheus | Datadog, New Relic | Self-hosted, sem custo adicional |

---

## 3. Checklist de Go-Live

### Infraestrutura (@devops)
- [ ] PostgreSQL: banco `notification_db` criado, migrations aplicadas
- [ ] Redis DB7: disponível e acessível
- [ ] RabbitMQ: exchanges e filas criadas (script `rabbitmq-setup.sh`)
- [ ] Secrets configurados em K3s (Resend API key, Evolution API key)
- [ ] Alertas configurados no Alertmanager (DLQ, delivery rate, latência)
- [ ] Health check do serviço respondendo (`/health` e `/health/ready`)

### Backend (@backend)
- [ ] Todos os consumers registrados e consumindo das filas corretas
- [ ] Deduplicação via Redis funcionando (teste manual)
- [ ] Fallback de canal funcionando (desligar WhatsApp → email enviado)
- [ ] Dead-letter queue configurada e processando falhas
- [ ] Endpoint `POST /api/v1/notifications/opt-out` acessível sem auth
- [ ] Webhooks dos providers configurados (Resend bounce, Meta Cloud status)

### Legal e Compliance (@legal)
- [ ] Tabela `notification_preferences` com campos LGPD completos
- [ ] Tabela `consent_audit_log` criada e imutável
- [ ] Templates WhatsApp submetidos à Meta (se usando Meta Cloud API)
- [ ] Link de unsubscribe em todos os templates de email
- [ ] Testado: opt-out via link bloqueia próximas notificações
- [ ] Política de privacidade do produto atualizada

### Qualidade (@qa)
- [ ] Cobertura ≥ 90% (`go test ./... -cover`)
- [ ] Testes de dedup passando (100% cobertura)
- [ ] Testes de opt-out passando
- [ ] Testes de fallback passando
- [ ] Teste de performance: p95 < 30s com 100 eventos simultâneos
- [ ] Testes de contrato: payload RabbitMQ validado

### Frontend (@frontend)
- [ ] Admin panel: histórico de notificações por cliente funcionando
- [ ] Botão "Reenviar" operacional
- [ ] Tela de preferências de canal pelo usuário final
- [ ] Página de unsubscribe sem autenticação
- [ ] Dashboard de métricas (delivery rate, DLQ, latência)

### Dados (@data)
- [ ] Todos os campos do schema `notifications` sendo populados
- [ ] `opened_at` sendo atualizado via webhook (email pixel/WhatsApp read receipt)
- [ ] Dashboard de métricas funcionando com dados reais
- [ ] Alertas de opt-out rate configurados

### IA (@ai)
- [ ] Feature flag `AI_PERSONALIZATION_ENABLED=false` por padrão (ativar pós-estabilização)
- [ ] Gemini API key configurada como secret
- [ ] Fallback testado: Gemini indisponível → template base enviado

---

## 4. Fases de Rollout

### Fase 0 (Pré-go-live, 2 semanas)
- Submeter templates WhatsApp à Meta (se usando Meta Cloud)
- Setup de infra (RabbitMQ, Redis, DB)
- Código base + testes

### Fase 1 (Go-live conservador, Semana 1)
- Apenas evento `payment.confirmed` ativo
- Monitorar delivery rate e DLQ
- Canal: apenas email (mais confiável)
- Objetivo: validar pipeline RabbitMQ → notification-service → provider

### Fase 2 (Expandir eventos, Semana 2-3)
- Ativar `class.reminder` e `order.ready`
- Ativar WhatsApp (Evolution API)
- Monitorar opt-out rate (alerta se > 2%)

### Fase 3 (Completo, Mês 2)
- Ativar `trial.expiring` e `payment.overdue`
- Ativar IA de personalização (feature flag)
- Ativar push notifications

### Fase 4 (Escala, M6)
- Avaliar migração Evolution → Meta Cloud API
- Ativar timing inteligente

---

## 5. Riscos Técnicos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Evolution API banido | Média | Alto | Abstrair interface; Meta Cloud API como backup |
| Gemini Flash Lite fora do ar | Baixa | Baixo | Fail-open → template base |
| Redis indisponível | Baixa | Médio | Fail-open dedup → aceitar duplicata |
| RabbitMQ redelivery em massa | Baixa | Médio | Dedup + idempotency no handler |
| Volume excede free tier Resend | Baixa M3 | Baixo | Alert em 2.500/mês → upgrade automático |
| DLQ crescendo sem controle | Baixa | Alto | Alert crítico + runbook @support/@devops |
