# PRD — notification-service

**Versão:** 0.1 (Planejado)  
**Data:** 2026-02-26  
**Stack:** Go + Fiber  
**Porta:** :3012

---

## 1. Visão

Serviço central de notificações multi-canal. Consome eventos de outros serviços via RabbitMQ e envia por email (Resend), WhatsApp (Evolution API) ou push (Expo).

**Importante:** O notification-service envia notificações *transacionais* (confirmação de pagamento, lembrete de aula). O **attend-agent** lida com *atendimento conversacional*. São complementares, não concorrentes.

---

## 2. Canais

| Canal | Provedor | Casos de uso |
|-------|---------|-------------|
| Email | Resend | Confirmação de pedido, recibo, NFS-e |
| WhatsApp | Evolution API | Lembrete de aula, vencimento de mensalidade |
| Push | Expo Push | Alertas no app mobile |

---

## 3. Eventos consumidos (RabbitMQ)

| Exchange | Evento | Notificação enviada |
|----------|--------|---------------------|
| checkout | order.completed | Email + WA confirmação de pagamento |
| scheduling | class.reminder | WA lembrete 24h antes da aula |
| checkout | payment.overdue | WA + email cobrança de inadimplência |
| scheduling | checkin.confirmed | Push confirmação de presença |

---

## 4. Schema

```sql
notifications (
    id UUID PRIMARY KEY,
    customer_id UUID,
    channel VARCHAR(20),
    template_type VARCHAR(50),
    payload JSONB,
    status VARCHAR(20),    -- pending | sent | failed
    provider_id VARCHAR(200),
    sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
)
```

**Deduplicação:** Redis DB7, chave `notif:{customer_id}:{template_type}:{event_id}`, TTL 24h.

---

## 5. Meta de Qualidade

- Testes unitários ≥70%
- Dead-letter queue para falhas após 3 retries
