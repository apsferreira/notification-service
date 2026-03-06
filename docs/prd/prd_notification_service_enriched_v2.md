# PRD Enriquecido — notification-service v2

**Versão:** 2.0 (Enriquecido pela cadeia de 11 agentes)  
**Data:** 2026-03-03  
**Status:** Pronto para implementação  
**Agente:** @pm — VEREDITO FINAL  
**Referências:** chain_01_data.md a chain_11_techlead.md

---

## Veredito do @pm

> Este PRD consolida os outputs de @data, @finance, @growth, @legal, @backend, @frontend, @devops, @ai, @qa, @support e @techlead em um documento único acionável. Ele substitui o PRD v0.1 e deve ser a fonte de verdade para implementação.
>
> **Decisão:** APROVAR para implementação com as especificações abaixo. Canal primário WhatsApp via Evolution API (MVP). Migração para Meta Cloud API em M6. IA opcional desde o início mas desativada por padrão.

---

## 1. Visão e Objetivos

### 1.1 Visão
Serviço central de notificações multi-canal que transforma eventos de negócio em comunicações que **aumentam receita, reduzem suporte e mantêm clientes engajados**, respeitando LGPD e com custo marginal próximo de zero no MVP.

### 1.2 North Star Metric
**≥ 95% dos eventos críticos entregues com sucesso em < 30 segundos** (definido por @data)

Eventos críticos:
- `payment.confirmed` — recibo de pagamento
- `class.reminder` — lembrete 24h antes da aula
- `order.ready` — pedido pronto
- `trial.expiring` — trial expirando em 48h

### 1.3 ROI Justificado (@finance)
| Benefício | Valor Mensal |
|-----------|-------------|
| Suporte evitado (recibos) | R$2.500 |
| No-show reduzido 60% (academia) | R$13.500 capacidade recuperada |
| Conversão trial → pago (+15 clientes) | R$1.500 MRR adicional |
| Inadimplência recuperada (40%) | R$600 MRR |
| **Total benefício** | **~R$18.100/mês** |
| **Custo total (M3)** | **~R$175/mês (~$35)** |
| **ROI** | **103x** |

---

## 2. Especificações Técnicas

### 2.1 Stack
```
Runtime:    Go 1.23 + Fiber v2
Database:   PostgreSQL (notification_db)
Cache:      Redis DB7 (deduplicação)
Queue:      RabbitMQ (consumers por tipo de evento)
Email:      Resend SDK
WhatsApp:   Evolution API (MVP) → Meta Cloud API (M6+)
Push:       Expo Push API
IA:         Gemini 2.0 Flash Lite (personalização seletiva, opcional)
Porta:      3012
```

### 2.2 Schema de Banco de Dados

```sql
-- Tabela principal
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID UNIQUE NOT NULL,
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,  -- email | whatsapp | push
    template_type   VARCHAR(50) NOT NULL,
    event_type      VARCHAR(50) NOT NULL,
    event_id        VARCHAR(100),          -- obrigatório para dedup
    payload         JSONB,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    provider_id     VARCHAR(200),
    error_code      VARCHAR(50),
    error_message   TEXT,
    attempt_count   INTEGER DEFAULT 0,
    queued_at       TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    opened_at       TIMESTAMPTZ,
    clicked_at      TIMESTAMPTZ,
    opted_out_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Preferências e consentimento LGPD
CREATE TABLE notification_preferences (
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,
    opted_in        BOOLEAN DEFAULT true,
    opted_in_at     TIMESTAMPTZ,
    opted_out_at    TIMESTAMPTZ,
    opt_in_source   VARCHAR(50),
    opt_out_source  VARCHAR(50),
    consent_text    TEXT,
    consent_version VARCHAR(20),
    ip_address      INET,
    user_agent      TEXT,
    PRIMARY KEY (customer_id, tenant_id, channel)
);

-- Audit log imutável (LGPD - INSERT ONLY)
CREATE TABLE consent_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,
    action          VARCHAR(20) NOT NULL,  -- opted_in | opted_out
    consent_text    TEXT,
    consent_version VARCHAR(20),
    ip_address      INET,
    performed_at    TIMESTAMPTZ DEFAULT NOW(),
    performed_by    VARCHAR(50)
);

-- Eventos de entrega (webhooks dos providers)
CREATE TABLE delivery_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID REFERENCES notifications(id),
    event           VARCHAR(50) NOT NULL,
    provider        VARCHAR(30) NOT NULL,
    provider_data   JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### 2.3 Eventos RabbitMQ Consumidos

| Exchange | Routing Key | Canal Principal | Fallback |
|----------|-------------|----------------|---------|
| checkout.events | payment.confirmed | Email + WhatsApp | Email (obrigatório) |
| checkout.events | payment.overdue | WhatsApp | Email |
| checkout.events | trial.expiring | WhatsApp | Email |
| scheduling.events | class.reminder | WhatsApp | Push, Email |
| scheduling.events | checkin.confirmed | Push | - |
| order.events | order.ready | WhatsApp | Push |
| table.events | waiter.requested | Push (garçom) | - |

### 2.4 API HTTP

```
POST /api/v1/notify              → envio direto (M2M, service token)
GET  /api/v1/notifications       → histórico (JWT, por customer_id)
POST /api/v1/notifications/opt-out → opt-out sem auth (token único)
GET  /api/v1/notifications/preferences/:customer_id → preferências
PATCH /api/v1/notifications/preferences/:customer_id → atualizar preferências
GET  /api/v1/notifications/metrics → métricas (admin)
GET  /health                     → health check
GET  /health/ready               → readiness check
GET  /metrics                    → Prometheus metrics
```

---

## 3. Decisões Arquiteturais (ADRs — @techlead)

| ADR | Decisão | Gatilho de Revisão |
|-----|---------|-------------------|
| Canal WhatsApp | Evolution API (MVP) | Volume > 2.000 conv/mês → migrar Meta Cloud |
| Retry policy | 3 tentativas: 5s/30s/120s (críticos: 5s/15s/60s) | DLQ crescendo > 5%/dia |
| Deduplicação | Redis SETNX, TTL 24h, fail-open | Redis instabilidade recorrente |
| IA | Gemini Flash Lite, opcional, bypass obrigatório | Custo > $50/mês ou latência > 4s |
| Provider abstraction | Interface WhatsAppProvider (swap por env var) | Migração para Meta Cloud API |

---

## 4. Requisitos Legais (LGPD — @legal)

### Obrigatórios para Go-Live:
1. **Opt-in explícito** para notificações comerciais/marketing
2. **Opt-out imediato** via link em emails + "SAIR" no WhatsApp
3. **consent_audit_log** com IP e user agent (INSERT ONLY)
4. **Templates Meta submetidos** 10 dias antes (se Meta Cloud API)
5. **Unsubscribe link** em todos os emails
6. **Webhook de bounce** (Resend) → opt-out automático

### Bases Legais por Tipo:
- Transacional (payment.confirmed, order.ready): execução de contrato → **não requer opt-in**
- Marketing (upsell, reativação): **requer opt-in explícito**
- Lembretes (class.reminder): legítimo interesse → opt-out disponível

---

## 5. Estratégia de Canais (@growth)

### Hierarquia de Canal por Template
| Template | Primário | Fallback 1 | Fallback 2 |
|---------|---------|-----------|-----------|
| payment.confirmed | Email* | WhatsApp | - |
| class.reminder | WhatsApp | Push | Email |
| trial.expiring | WhatsApp | Email | - |
| order.ready | WhatsApp | Push | - |
| payment.overdue | WhatsApp | Email | - |

*Email é obrigatório para payment.confirmed (prova de transação)

### Janela de Opt-In (@growth + @legal)
- **Email:** No signup (always-on para transacional)
- **WhatsApp:** Pós-primeiro pagamento ("receber lembretes pelo WhatsApp?")
- **Push:** Na instalação do app mobile

---

## 6. Inteligência Artificial (@ai)

### Feature Flag: `AI_PERSONALIZATION_ENABLED` (default: false em M1)

**Templates que recebem personalização (ativar em M2):**
1. `trial.expiring` — maior ROI (@finance: +R$1.500 MRR por 15 conversões)
2. `payment.overdue` — tom adequado aumenta recovery rate

**Modelo:** Gemini 2.0 Flash Lite ($0,075/1M input tokens)  
**Custo estimado M3:** $0,26/mês (desprezível)  
**Timeout:** 4s (fallback para template base se exceder)

**Regra absoluta:** `payment.confirmed` e `order.ready` NUNCA chamam IA.

**Timing Inteligente (M3):** Calcular melhor horário de envio baseado em `opened_at` histórico (@data coleta desde o dia 1).

---

## 7. Infraestrutura (@devops)

### Kubernetes (K3s — namespace: shared-services)
- **Replicas:** 2 (HA mínima)
- **Resources:** 128Mi/100m → 256Mi/500m
- **Rolling update:** 1 pod por vez

### Redis: DB7 (dedup, TTL 24h)
### RabbitMQ: exchanges fanout, DLQ configurada

### Alertas Críticos:
| Condição | Severidade |
|----------|-----------|
| DLQ > 50 mensagens | Critical |
| Delivery rate < 90% por 5min | Warning |
| p95 latência > 30s | Warning |
| WhatsApp opt-out rate > 2% | Warning |
| Service down | Critical |

---

## 8. Qualidade (@qa)

**Meta de cobertura:** 90% (acima dos 70% do PRD v0.1 — justificado pelo ROI de R$18k/mês)

**Casos de teste obrigatórios:**
1. Deduplicação: mesmo event_id → apenas 1 notificação (100% cobertura)
2. Opt-out: cliente opted_out não recebe mesmo com evento válido
3. Fallback: WhatsApp falha → email enviado
4. Dead-letter: max retries → DLQ, status=failed na DB
5. IA fallback: Gemini down → template base enviado, sem falha da notificação
6. Performance: p95 < 30s com 100 eventos simultâneos

**Gates de CI:**
- Cobertura ≥ 90% em cada PR
- Testes de contrato (payload RabbitMQ) em cada PR
- Integration tests: weekly

---

## 9. Frontend (@frontend)

### Admin Panel
- Dashboard: delivery rate por canal, latência p50/p95/p99, DLQ size
- Histórico de notificações por cliente (filtros: canal, status, data)
- Botão "Reenviar" por notificação
- Templates manager com status de aprovação Meta
- Painel de custo (consumo vs. budget $200/mês)

### Usuário Final
- Tela de preferências: toggle por canal e por tipo de notificação
- Notificações transacionais (payment, order): não desativáveis
- Página de unsubscribe sem autenticação (via token único)

---

## 10. Suporte (@support)

### Playbooks Documentados:
1. **Cliente não recebeu** → verificar status na DB, reenviar se failed
2. **Opt-out** → processar imediatamente (prioridade legal), confirmar ao cliente
3. **Notificação duplicada** → verificar event_id no Redis, abrir bug se dedup falhou
4. **WhatsApp ban** → fallback para email ativo, @devops novo número

### SLA de Suporte:
- Opt-out: processar em < 1 hora
- Não recebi pagamento: reenviar em < 2 horas
- Bug (duplicata/dados errados): escalação imediata para @backend

---

## 11. Cronograma de Implementação

### Fase 0 — Pré-go-live (2 semanas)
- [ ] Setup infra (DB, Redis, RabbitMQ)
- [ ] Código base: consumers, dispatch engine, dedup, providers
- [ ] Testes (90% cobertura)
- [ ] Submeter templates WhatsApp (se Meta Cloud)

### Fase 1 — Go-live Conservador (Semana 1-2)
- [ ] Apenas `payment.confirmed` ativo, canal email
- [ ] Validar pipeline end-to-end
- [ ] Alertas funcionando

### Fase 2 — Expansão (Semana 3-4)
- [ ] Ativar `class.reminder` e `order.ready`
- [ ] Ativar WhatsApp (Evolution API)
- [ ] Opt-in WhatsApp no fluxo de pagamento (@growth)

### Fase 3 — Completo (M2)
- [ ] Ativar `trial.expiring` e `payment.overdue`
- [ ] Ativar push notifications
- [ ] Frontend admin panel completo
- [ ] Ativar IA de personalização (feature flag)

### Fase 4 — Escala (M6)
- [ ] Avaliar migração Evolution → Meta Cloud API
- [ ] Ativar timing inteligente
- [ ] Review de ROI vs. projeção @finance

---

## 12. Métricas de Sucesso

| Métrica | Linha de Base | Meta M3 | Meta M12 |
|---------|--------------|---------|---------|
| North Star (< 30s) | 0% (não existe) | ≥ 95% | ≥ 99% |
| Delivery rate email | - | ≥ 98% | ≥ 99% |
| Delivery rate WhatsApp | - | ≥ 92% | ≥ 97% |
| Open rate WhatsApp | - | ≥ 75% | ≥ 80% |
| Opt-out rate WhatsApp | - | < 2% | < 1% |
| Tickets "não recebi" | ~10%/transação | < 1% | < 0,5% |
| Trial → Paid (com notif) | - | ≥ 20% | ≥ 25% |
| Custo total | R$0 (sem serviço) | < R$175/mês | < R$600/mês |

---

## 13. Dependências e Bloqueadores

| Dependência | Status | Ação Necessária |
|------------|--------|----------------|
| shared-infra (RabbitMQ, Redis, PostgreSQL) | ✅ Existente | Apenas configurar DB e filas |
| Evolution API | ✅ Existente (shared-infra) | Apenas credenciais |
| Resend API key | ❌ Pendente | Criar conta e gerar key |
| Meta Cloud API templates | ❌ Bloqueador se usar Meta | Submeter 10 dias antes |
| Gemini API key | ❌ Pendente | Obter no Google AI Studio |
| Checkout-service publicando eventos | 🔄 Em desenvolvimento | Validar payload do evento |
| Scheduling-service publicando class.reminder | 🔄 Em desenvolvimento | Validar payload + timing 24h |

---

## Aprovação Final do @pm

**APROVADO para implementação.**

Prioridade de início: **ALTA** — ROI de 103x justifica implementação imediata.

O notification-service desbloqueia valor em TODOS os outros serviços do ecossistema. Cada produto vertical (jiu-jitsu-academy, food-marketplace, restaurant-qr) tem dependência funcional e de receita neste serviço.

Implementar em ordem: consumer base → dedup → email provider → testes → WhatsApp provider → frontend → IA.
