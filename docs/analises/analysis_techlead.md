# Analysis Tech Lead — notification-service

**Agente:** @techlead  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. ADRs (Architecture Decision Records)

### ADR-NT-001: WhatsApp via Meta Cloud API (não Evolution API)

**Status:** Decidido (CEO)  
**Contexto:** PRD original mencionava Evolution API. CEO reverteu para Meta Cloud API.  
**Decisão:** Usar exclusivamente Meta Cloud API.  
**Consequências:**
- PRO: Oficial, estável, sem risco de ban por uso não-oficial
- PRO: Webhooks nativos de status (delivered, read)
- CON: Templates precisam de aprovação prévia (delay de 1-7 dias antes do launch)
- CON: Custo por conversa (~$0,06/conversa no Brasil)
- Ação: Criar templates e submeter para aprovação Meta ANTES de começar o desenvolvimento

### ADR-NT-002: Consumer RabbitMQ + Dead Letter Queue

**Status:** Adotado  
**Contexto:** Precisa processar eventos de múltiplos serviços com garantia de entrega.  
**Decisão:** Consumer persistente com DLQ após 3 retries.  
**Alternativa considerada:** Polling de tabela de eventos (outbox pattern) — descartado pela complexidade adicional dado que RabbitMQ já existe na infra.  
**Consequências:**
- DLQ requer monitoramento e processo de replay manual quando necessário
- Consumer deve ser idempotente (deduplicação Redis)

### ADR-NT-003: Deduplicação com Redis SetNX (não banco de dados)

**Status:** Adotado  
**Contexto:** Garantir idempotência de envios mesmo com redelivery RabbitMQ.  
**Decisão:** Redis SetNX atômico com TTL 24h.  
**Alternativa considerada:** Unique constraint na tabela `notifications` com (customer_id, event_id) — descartada porque: (1) adiciona latência de I/O de banco, (2) gera exceptions no banco para o fluxo normal de dedup.  
**Consequências:** Redis DB7 é dependência crítica. Se Redis cair, risco de duplicatas. Aceitável dado que shared-infra-01 tem Redis com persistência RDB.

### ADR-NT-004: Templates em Banco (não código)

**Status:** Adotado (REQ-NT-01)  
**Contexto:** Templates transacionais precisam de flexibilidade sem redeploy.  
**Decisão:** Tabela `notification_templates` com body_template e variáveis.  
**Consequências:** Precisa de seed inicial com templates. Admin pode atualizar templates sem deploy.

### ADR-NT-005: Stack Go + Fiber (consistente com ecossistema)

**Status:** Adotado  
**Contexto:** Ecossistema IIT usa Go como backend padrão.  
**Observação:** O notification-service é principalmente um worker (consumer), não uma API HTTP-heavy. Fiber é usado apenas para healthcheck, webhook receiver e admin endpoints.

---

## 2. Sequência de Implementação

### Fase 1 — Foundation (Sprint 1, ~5 dias)

```
[ ] 1. Setup do repositório e estrutura de diretórios
[ ] 2. Migrations: notification_templates, notifications, notification_preferences
[ ] 3. Conexões: PostgreSQL, Redis DB7, RabbitMQ
[ ] 4. Health/Ready endpoints
[ ] 5. CI pipeline básico (build + test)
```

### Fase 2 — Core Email (Sprint 2, ~4 dias)

```
[ ] 6. Resend adapter (envio + webhook)
[ ] 7. Template resolver (PostgreSQL lookup + variable substitution)
[ ] 8. Deduplication service (Redis SetNX)
[ ] 9. Consumer RabbitMQ básico (ack/nack)
[ ] 10. Processamento de evento: checkout.confirmed → email
[ ] 11. Seed de templates de email
[ ] 12. Testes unitários (dedup, template resolver)
```

### Fase 3 — WhatsApp (Sprint 3, ~4 dias)

```
[ ] 13. Meta Cloud API adapter
[ ] 14. Webhook receiver (Meta → atualiza status no banco)
[ ] 15. Processamento: scheduling.class.reminder → WhatsApp
[ ] 16. Retry com backoff exponencial
[ ] 17. DLQ configurada
[ ] 18. Seed de templates WhatsApp (após aprovação Meta)
[ ] 19. Testes de integração com mock da Meta API
```

### Fase 4 — Observabilidade + Opt-Out (Sprint 4, ~3 dias)

```
[ ] 20. Prometheus metrics
[ ] 21. Endpoint /metrics
[ ] 22. Opt-out handler (WhatsApp incoming messages)
[ ] 23. notification_preferences CRUD
[ ] 24. Alertas Prometheus configurados
[ ] 25. Deploy em K3s
```

**Total estimado: ~16 dias úteis (4 sprints de 1 semana)**

---

## 3. O que Desbloqueia o Brio

### Dependência Crítica

O Brio (attend-agent) atualmente usa Telegram direto para escaladas. A migração para notification-service desbloqueia:

1. **Multi-canal para escaladas** — escalada pode ir por WhatsApp (melhor UX) em vez de apenas Telegram
2. **Log centralizado** — todas as comunicações do Brio registradas em `notifications` table
3. **Templates para o Brio** — Brio pode publicar evento `brio.escalation` → notification-service entrega no canal correto

### Evento que o Brio Precisa Publicar

```json
{
  "exchange": "brio",
  "routing_key": "brio.escalation",
  "payload": {
    "customer_id": "uuid",
    "customer_name": "João",
    "customer_phone": "+5571999999999",
    "conversation_summary": "Cliente com dúvida sobre pagamento",
    "escalation_channel": "whatsapp",  // ou "telegram"
    "escalated_to": "admin"
  }
}
```

### Template Necessário

```
TIPO: brio.escalation | CANAL: whatsapp
TEXTO: "🚨 Escalada do Brio — {{1}} aguarda atendimento.
Resumo: {{2}}
Contato: {{3}}"
```

**IMPORTANTE:** Para MVP do Brio (v1.0), manter Telegram direto para o Antonio. notification-service habilita a migração em v1.1. Não bloquear o Brio por isso.

---

## 4. Pontos de Atenção Técnica

1. **Webhook da Meta precisa de HTTPS** — garantir certificado TLS no cluster (cert-manager já deve estar configurado)
2. **Warmup de templates WhatsApp** — submeter templates para aprovação na semana 1, antes de qualquer desenvolvimento; templates rejeitados precisam de ciclo de revisão
3. **Segurança do payload** — não logar `payload` completo se contiver PII (telefone, email); logar apenas `event_id` e `template_type`
4. **Idempotência do consumer** — o consumer DEVE ser idempotente mesmo sem Redis (fallback: verificar `SELECT 1 FROM notifications WHERE event_id = $1`)
5. **Graceful shutdown** — ao receber SIGTERM, terminar processamento da mensagem corrente antes de fechar conexão RabbitMQ
