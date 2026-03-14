# chain_01_data.md — @data
# Análise de Dados: notification-service

**Agente:** @data  
**Data:** 2026-03-03  
**Papel:** Driver primário de métricas e coleta de dados para toda a cadeia

---

## 1. North Star Metric

> **% de eventos críticos entregues com sucesso em < 30 segundos**

Definição de "evento crítico":
- `payment.confirmed` → recibo de pagamento
- `class.reminder` → lembrete de aula (24h antes)
- `order.ready` → pedido pronto
- `trial.expiring` → trial expirando (em 48h)

Meta inicial: **≥ 95% em < 30s** em M3. Meta de maturidade: **≥ 99% em < 30s** em M12.

---

## 2. Métricas por Canal

### 2.1 Taxa de Entrega (Delivery Rate)
| Canal | Fórmula | Meta M3 | Meta M12 |
|-------|---------|---------|---------|
| Email (Resend) | enviados_ok / tentativas | ≥ 98% | ≥ 99% |
| WhatsApp (Evolution API) | enviados_ok / tentativas | ≥ 92% | ≥ 97% |
| Push (Expo) | enviados_ok / tentativas | ≥ 85% | ≥ 90% |

> WhatsApp tem menor delivery rate inicial por falhas de template ou número inválido.
> Push tem a menor taxa por opt-in mais restritivo em iOS.

### 2.2 Taxa de Abertura (Open Rate)
| Canal | Benchmark Mercado | Meta IIT |
|-------|------------------|---------|
| Email | 20–30% | 35% (transacional tem abertura maior) |
| WhatsApp | 70–90% | 80% |
| Push | 5–15% | 10% |

> WhatsApp é o canal de maior abertura — **driver para @finance e @growth priorizarem WhatsApp**.

### 2.3 Taxa de Conversão Pós-Notificação
| Tipo de Notificação | Conversão Esperada | KPI |
|--------------------|--------------------|-----|
| `class.reminder` | 60% menos no-show | checkin_rate pós-envio |
| `payment.confirmed` | N/A (confirmação) | redução tickets suporte |
| `trial.expiring` | 15-25% upgrade pago | conversion_rate |
| `payment.overdue` | 30-50% pagam em 24h | recovery_rate |
| `order.ready` | N/A (ação imediata) | customer_satisfaction |

### 2.4 Opt-Out Rate
| Canal | Alerta (acima de) | Crítico (acima de) |
|-------|------------------|--------------------|
| Email | 0.5% | 1% (risco de spam score) |
| WhatsApp | 2% | 5% |
| Push | 5% | 10% |

> Opt-out rate alto = problema de relevância ou frequência. Acionar @growth se ultrapassar alerta.

---

## 3. Schema de Coleta — Desde o Dia 1

```sql
-- Tabela principal (expandida do PRD original)
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID UNIQUE NOT NULL,  -- ID externo para dedup
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,  -- email | whatsapp | push
    template_type   VARCHAR(50) NOT NULL,  -- payment.confirmed | class.reminder | etc.
    event_type      VARCHAR(50) NOT NULL,  -- evento RabbitMQ original
    event_id        VARCHAR(100),          -- ID do evento no RabbitMQ (dedup)
    payload         JSONB,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    -- pending | queued | sent | delivered | failed | bounced | opted_out
    provider_id     VARCHAR(200),          -- ID retornado pelo Resend/Evolution/Expo
    error_code      VARCHAR(50),           -- código de erro se falhou
    error_message   TEXT,
    attempt_count   INTEGER DEFAULT 0,
    queued_at       TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,           -- confirmado pelo provider (quando disponível)
    opened_at       TIMESTAMPTZ,           -- via pixel tracking (email) ou webhook
    clicked_at      TIMESTAMPTZ,           -- link tracking
    opted_out_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Índices críticos para analytics
CREATE INDEX idx_notifications_customer ON notifications(customer_id);
CREATE INDEX idx_notifications_channel_status ON notifications(channel, status);
CREATE INDEX idx_notifications_event_type ON notifications(event_type, created_at);
CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);

-- Tabela de opt-out (LGPD)
CREATE TABLE notification_preferences (
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,
    opted_in        BOOLEAN DEFAULT true,
    opted_in_at     TIMESTAMPTZ,
    opted_out_at    TIMESTAMPTZ,
    opt_in_source   VARCHAR(50),           -- signup | manual | sms | whatsapp
    opt_out_source  VARCHAR(50),           -- manual | unsubscribe_link | whatsapp_stop
    PRIMARY KEY (customer_id, tenant_id, channel)
);

-- Log de eventos de entrega (webhook providers)
CREATE TABLE delivery_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID REFERENCES notifications(id),
    event           VARCHAR(50) NOT NULL,  -- sent | delivered | opened | bounced | failed
    provider        VARCHAR(30) NOT NULL,
    provider_data   JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 4. Eventos RabbitMQ — Mapeamento Completo

| Exchange | Routing Key | Payload Mínimo | Notificações |
|----------|-------------|----------------|-------------|
| `checkout.events` | `payment.confirmed` | order_id, customer_id, amount, tenant_id | Email recibo + WhatsApp confirmação |
| `checkout.events` | `payment.overdue` | subscription_id, customer_id, due_date | WhatsApp + email cobrança |
| `scheduling.events` | `class.reminder` | slot_id, customer_id, class_name, starts_at | WhatsApp 24h antes |
| `scheduling.events` | `checkin.confirmed` | slot_id, customer_id | Push confirmação |
| `order.events` | `order.ready` | order_id, customer_id, restaurant_name | WhatsApp + Push |
| `checkout.events` | `trial.expiring` | customer_id, expires_at, plan_name | WhatsApp + email |
| `scheduling.events` | `class.cancelled` | slot_id, customer_ids[], reason | WhatsApp aviso |
| `table.events` | `waiter.requested` | table_id, session_id, tenant_id | Push garçom |

---

## 5. Dashboards e Alertas Recomendados

### Dashboards (admin panel)
1. **Delivery Overview** — taxa de entrega por canal, últimas 24h/7d/30d
2. **Event Pipeline** — eventos recebidos vs. notificações enviadas (funil)
3. **Latência** — distribuição p50/p95/p99 do tempo evento→entrega
4. **Opt-out Trends** — taxa de opt-out por canal e por tipo de notificação
5. **Error Analysis** — erros por provider, por error_code

### Alertas (operacionais)
| Condição | Ação |
|----------|------|
| Delivery rate < 90% em 5min | Alert Slack/Telegram @devops |
| Dead-letter queue > 50 msgs | Alert crítico |
| Latência p95 > 30s | Alert |
| Opt-out rate > 2% em 1h | Alert @growth |

---

## 6. O Que @finance e os Demais Precisam Saber

### Para @finance:
- **Volume base**: cada evento gera 1-2 notificações. Com 10 tenants ativos, estimativa M3: ~5.000 notificações/mês
- **Canal mais barato**: Push (Expo) = gratuito. Email via Resend = gratuito até 3k/mês
- **Canal com custo**: WhatsApp via Meta Cloud API = $0,06/conversa (ou Evolution API self-hosted = custo de infra apenas)
- **Dado crítico**: `delivered_at` e `opened_at` permitem calcular ROI real por notificação
- **Fallback econômico**: se WhatsApp falha, cair para email (gratuito) reduz custo de retry

### Para @backend:
- Redis DB7 para dedup: chave `notif:{customer_id}:{template_type}:{event_id}` TTL 24h
- Campo `event_id` no payload RabbitMQ é obrigatório para dedup correto
- Status `delivered` só atualizado via webhook do provider — não confundir com `sent`

### Para @growth:
- WhatsApp tem open rate 80% vs email 35% — priorizar WhatsApp para ações de conversão
- `trial.expiring` tem maior potencial de conversão pós-notificação (15-25%)
- Dados de `clicked_at` permitem A/B test de CTAs

### Para @legal:
- `notification_preferences` é a fonte de verdade de consentimento (LGPD)
- `opted_in_at` e `opt_in_source` são obrigatórios desde o registro
- Opt-out deve ser processado em < 1 hora (meta: imediato)

### Para @qa:
- Testar dedup: enviar mesmo event_id duas vezes → apenas 1 notificação
- Testar opt-out: cliente opted_out não recebe mesmo com evento válido
- Testar latência: 95% dos eventos críticos em < 30s

---

## 7. KPIs de Saúde do Serviço (Semana 1)

Desde o primeiro deploy, coletar:
- [ ] Total eventos recebidos por tipo
- [ ] Taxa de sucesso por canal
- [ ] Latência média evento → entrega
- [ ] Contagem de dead-letter por dia
- [ ] Opt-out count por canal
- [ ] Erros por provider e error_code

> Sem esses dados, @finance não pode calcular ROI, @growth não pode otimizar canais, e @techlead não tem base para ADRs.
