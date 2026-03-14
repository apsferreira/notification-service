# PRD — notification-service
**Versão:** 1.0 enriched  
**Data:** 2026-03-03  
**Owner:** @pm  
**Status:** Em revisão  
**Prioridade:** 🟠 P1

---

## 1. Executive Summary + Veredito

O **notification-service** é o hub central de comunicações transacionais do Instituto Itinerante (IIT). Atua como consumer de eventos RabbitMQ e entrega mensagens por **email** (Resend), **WhatsApp** (Meta Cloud API) e **push** (Expo) para alunos e administradores.

### Veredito: ✅ Construir agora

**Por quê urgente:**
- Checkout e agendamento já existem; alunos não recebem confirmações — UX quebrada
- Lembretes de aula reduzem absenteísmo (~20% típico em academias)
- Sem notification-service, o Brio (atendimento) não pode migrar do Telegram para WhatsApp
- Volume atual (~500 notif/mês) cabe no free tier do Resend — custo zero para MVP

**Escopo MVP:** Email transacional + WhatsApp via Meta Cloud API. Push em P2.  
**Prazo estimado:** ~16 dias úteis (4 sprints de 1 semana).  
**Bloqueio crítico:** Templates WhatsApp precisam de aprovação Meta (1–7 dias úteis) — submeter na semana 1, antes de qualquer desenvolvimento.

---

## 2. Papel no Ecossistema

### Posicionamento

O notification-service é **passivo em relação a eventos** (consumer, não produtor) e **ativo em relação a canais externos** (email, WhatsApp, push). Ele não tem lógica de negócio — apenas transforma eventos em notificações.

```
checkout-service ──┐
scheduling-service ─┼──► RabbitMQ ──► notification-service ──► Email (Resend)
membership-service ─┤                                        ──► WhatsApp (Meta Cloud API)
brio (v1.1) ────────┘                                        ──► Push (Expo, P2)
```

### Eventos Consumidos

| Routing Key | Gatilho | Canal(is) |
|---|---|---|
| `checkout.confirmed` | Pagamento confirmado | Email + WhatsApp |
| `checkout.payment.overdue` | Pagamento em atraso | Email + WhatsApp |
| `scheduling.class.reminder` | 24h antes da aula | WhatsApp (prioridade) |
| `scheduling.checkin.confirmed` | Check-in realizado | Email |
| `trial.expiring` | Trial vence em 48h | Email + WhatsApp |
| `brio.escalation` *(v1.1)* | Escalada do Brio | WhatsApp/Telegram |

### Quem Consome os Dados

- **Alunos** — recebem notificações transacionais no canal preferido
- **Admin (Antonio)** — recebe escaladas do Brio (MVP via Telegram direto; v1.1 via notification-service)
- **Times de dados** — tabela `notifications` como fonte de verdade de deliverability
- **Plataforma** — feedback de delivery via webhooks Resend/Meta atualiza status em tempo real

### Relação com o Brio

O Brio (attend-agent) faz **atendimento conversacional**; o notification-service faz **notificações de sistema**. São complementares, não concorrentes. Para o MVP do Brio (v1.0), manter Telegram direto para o Antonio. Migração para notification-service ocorre em v1.1, desbloqueando multi-canal nas escaladas.

---

## 3. Dados e Métricas

*@data driver: benchmarks de mercado Brasil B2C transacional 2024-2025*

### Benchmarks de Deliverability

| Canal | Métrica | Mercado Brasil | Meta IIT |
|---|---|---|---|
| **Email** | Delivery rate | 97–99% | ≥98% |
| **Email** | Open rate (transacional) | 45–65% | ≥50% |
| **Email** | Bounce rate (hard) | <0,5% | <0,3% |
| **Email** | Spam complaint rate | <0,08% | <0,05% |
| **WhatsApp** | Delivery rate | 95–98% | ≥95% |
| **WhatsApp** | Open rate | 85–95% | ≥85% |
| **WhatsApp** | Read rate em 5 min | 70–80% | — |

> **Insight estratégico:** WhatsApp tem 10x mais abertura que email para lembretes de aula. Priorizar WhatsApp para `scheduling.class.reminder`. Email é adequado para recibos e documentação.

### Volume Estimado MVP

- **Alunos ativos estimados:** 50–200 para MVP
- **Eventos/mês estimados:** ~500 notificações/mês
- **Resend free tier:** 3.000/mês (100/dia) — suficiente para MVP
- **Meta Cloud API:** ~$0,0625/conversa de utilidade no Brasil → custo ≤ $30/mês com 200 alunos

### Fatores Críticos de Deliverability (Email)

1. **SPF + DKIM + DMARC** — obrigatório para evitar spam
2. **Domínio dedicado** — `noreply@institutoitinerante.com.br`
3. **Resend gerencia warmup de IP**
4. **Latência:** p50 < 2s, p99 < 10s após evento
5. **Unsubscribe em 1 clique** — obrigatório Google/Yahoo desde 2024

### Templates WhatsApp Validados

Os seguintes templates devem ser submetidos à Meta para aprovação antes do launch:

| Template | Categoria Meta | Uso |
|---|---|---|
| `class_reminder` | UTILITY | Lembrete 24h antes da aula |
| `payment_confirmed` | UTILITY | Confirmação de pagamento |
| `subscription_expiring` | UTILITY | Aviso de vencimento 48h |
| `opt_out_confirmation` | UTILITY | Confirmação de cancelamento |

---

## 4. Canais e Custos

*@finance: análise comparativa de provedores*

### Decisão do CEO: WhatsApp = Meta Cloud API

> **⚠️ MANDATO CEO:** Não usar Evolution API. WhatsApp exclusivamente via Meta Cloud API.

**Rationale técnico para o mandato:**
- Meta Cloud API é a API oficial — sem risco de ban por uso não autorizado
- Webhooks nativos de status (`sent`, `delivered`, `read`, `failed`)
- Templates obrigatórios = controle de qualidade embutido
- SLA e suporte oficial disponíveis

### Comparativo de Custos

| Provedor | Canal | Plano/Custo | Observações |
|---|---|---|---|
| **Resend** | Email | Free: 3k/mês; $20/mês p/ 50k | Free tier suficiente para MVP |
| **Meta Cloud API** | WhatsApp | ~$0,0625 USD/conversa (BR) | Pago por conversa, não por mensagem |
| **Expo Push** | Push (mobile) | Gratuito até 1k/mês | P2 — não no MVP |
| ~~Evolution API~~ | ~~WhatsApp~~ | — | **❌ Proibido pelo CEO** |

### Custo Total Estimado — MVP

| Cenário | Volume/mês | Custo/mês |
|---|---|---|
| MVP lançamento | ~500 notif | $0 (free tiers) |
| Crescimento 200 alunos | ~2k notif | ~$12 (só Meta WA) |
| Escala 500 alunos | ~5k notif | ~$30 (Meta) + $20 (Resend) |

### Variáveis de Ambiente Necessárias

```
RESEND_API_KEY
RESEND_FROM_EMAIL=noreply@institutoitinerante.com.br
META_WA_PHONE_NUMBER_ID
META_WA_ACCESS_TOKEN
META_WA_API_VERSION=v18.0
```

---

## 5. Funcionalidades MVP — P0/P1/P2

### P0 — Bloqueia Deploy (deve estar 100% antes do go-live)

| ID | Funcionalidade | Descrição |
|---|---|---|
| REQ-NT-01 | Templates por evento | Tabela `notification_templates` — sem hardcode de mensagens |
| REQ-NT-02 | Deduplicação | Redis SetNX TTL 24h — previne double-send em redelivery RabbitMQ |
| REQ-NT-03 | Log de envios | Tabela `notifications` com status: `pending/sent/delivered/failed/opted_out` |
| REQ-NT-04 | Retry com backoff | 3 tentativas, backoff 1s→4s→16s; erros permanentes (4xx) não retentar |
| P0-NT-A | Consumer RabbitMQ | Binding para eventos de checkout, scheduling, trial |
| P0-NT-B | Email via Resend | Adapter com webhook de status (`delivered`, `bounced`, `complained`) |
| P0-NT-C | WhatsApp via Meta | Adapter com templates aprovados e webhook de status |
| P0-NT-D | Opt-out funcional | Verificação pré-envio; handler de "PARAR/CANCELAR" no WhatsApp |
| P0-NT-E | DLQ configurada | Dead Letter Queue após 3 falhas; alerta se DLQ > 10 mensagens |
| P0-NT-F | Health/Ready endpoints | `/health`, `/ready`, `/metrics` (Prometheus) |

### P1 — Importante (lançar em até 2 semanas pós-MVP)

| ID | Funcionalidade | Descrição |
|---|---|---|
| P1-NT-A | Admin de templates | Endpoint para CRUD de templates sem redeploy |
| P1-NT-B | Dashboard de delivery | Métricas de delivery rate por canal em tempo real |
| P1-NT-C | Replay de DLQ | Processo para reprocessar mensagens da DLQ manualmente |
| P1-NT-D | Webhook Resend | Atualizar status `delivered`/`bounced` no banco via webhook |
| P1-NT-E | Push (Expo) | Adapter para notificações push no app mobile |

### P2 — Backlog Futuro

| ID | Funcionalidade | Descrição |
|---|---|---|
| P2-NT-A | Brio escalation | Migrar escaladas do Telegram para notification-service (v1.1) |
| P2-NT-B | Preferências por evento | Aluno escolhe canal por tipo de notificação |
| P2-NT-C | A/B testing de templates | Teste de variantes de mensagem com tracking de open rate |
| P2-NT-D | Agendamento de envio | Enviar notificação em horário específico (não imediato) |
| P2-NT-E | Relatório mensal | Sumário de métricas de comunicação para o admin |

---

## 6. Compliance LGPD

*@legal: Lei 13.709/2018*

### Base Legal por Tipo de Notificação

| Notificação | Base Legal (Art. 7º LGPD) | Precisa Opt-In? |
|---|---|---|
| Confirmação de pagamento | Execução de contrato (inc. V) | Não |
| Lembrete de aula | Legítimo interesse (inc. IX) | Não (mas opt-out obrigatório) |
| Vencimento de mensalidade | Execução de contrato (inc. V) | Não |
| Promoções/marketing | Consentimento (inc. I) | **Sim** (fora do escopo MVP) |

> **Simplificação:** O notification-service IIT opera exclusivamente com notificações transacionais — não envia marketing. Isso simplifica substancialmente a gestão legal.

### Opt-In para WhatsApp (exigido pela Meta)

Meta exige opt-in explícito para mensagens business. Implementar no cadastro do aluno:

```
Checkbox (obrigatório marcar): 
"✅ Aceito receber lembretes de aula e comunicados via WhatsApp"
→ Registrar: opted_in=true, channel='whatsapp', timestamp, IP
```

### Opt-Out — Implementação Obrigatória

| Canal | Mecanismo | Prazo de Execução |
|---|---|---|
| **Email** | Link unsubscribe no rodapé de TODOS os emails (válido ≥30 dias) | Imediato |
| **WhatsApp** | Responder "PARAR", "CANCELAR" ou "STOP" → opt-out automático | Imediato |
| **Push** | Configuração nativa do sistema operacional | N/A (SO gerencia) |

### Schema de Preferências

```sql
notification_preferences (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    channel VARCHAR(20),         -- 'email', 'whatsapp', 'push'
    event_category VARCHAR(50),  -- 'transactional', 'reminder', 'marketing'
    opted_in BOOLEAN DEFAULT true,
    opted_out_at TIMESTAMP,
    opt_out_reason VARCHAR(200),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(customer_id, channel, event_category)
)
```

### Retenção de Dados

- **Logs de notificações (`notifications`):** reter por 24 meses (resolução de disputas)
- **Dados de opt-out:** reter indefinidamente (prova de conformidade)
- **PII em logs:** **não logar payload completo em plaintext**; usar apenas `event_id` como referência

### Checklist de Conformidade

- [ ] Política de Privacidade publicada em `institutoitinerante.com.br/privacidade`
- [ ] Mecanismo de opt-out funcional para cada canal
- [ ] Logs sem dados pessoais em plaintext
- [ ] Registro de base legal por tipo de notificação
- [ ] Processo de resposta a requisições de titulares (prazo legal: 15 dias)
- [ ] DPO ou responsável identificado (pode ser o fundador para pequeno porte)
- [ ] Opt-in WhatsApp coletado no cadastro com timestamp e IP

### Política Anti-Spam Meta Cloud API

- Monitorar: se >2% dos destinatários bloquearem o número → WABA vai para revisão
- Nunca enviar mesma mensagem mais de uma vez (deduplicação Redis atende)
- Templates sem links encurtados (usar domínio próprio)
- Máximo 1 lembrete por aula por aluno

---

## 7. KPIs e North Star

### North Star Metric

> **% de alunos que recebem notificação relevante no momento certo**

Operacionalizado como: `(notificações entregues / notificações tentadas) × 100`  
Meta: ≥95% ao longo de qualquer janela de 7 dias.

### KPIs Operacionais

| KPI | Fórmula | Meta | Alerta |
|---|---|---|---|
| **Delivery Rate Email** | delivered/sent | ≥98% | <95% |
| **Delivery Rate WhatsApp** | delivered/sent | ≥95% | <90% |
| **Open Rate Email** | opened/delivered | ≥50% | <35% |
| **Open Rate WhatsApp** | read/delivered | ≥85% | <70% |
| **Latência p99** | tempo evento→envio | <10s | >30s |
| **Taxa de Falha** | failed/total | <2% | >5% |
| **Retry Success Rate** | recovered/retried | ≥80% | <60% |
| **DLQ Volume** | msgs na DLQ | <5/dia | >10 acumulado |
| **Opt-Out Rate Email** | unsub/sent | <0,3% | >1% |
| **Spam Complaint Rate** | complained/sent | <0,05% | >0,08% |

### Métricas de Negócio (Proxy)

| Métrica | Hipótese | Como Medir |
|---|---|---|
| Redução de absenteísmo | Lembrete 24h antes → -20% faltas | Comparar check-ins antes/depois |
| Redução de inadimplência | Aviso vencimento → +X% renovações no prazo | Comparar antes/depois do launch |
| CSAT pós-notificação | Alunos informados → satisfação maior | NPS survey trimestral |

### Dashboard Prometheus (Grafana)

Métricas expostas em `/metrics`:
- `notifications_sent_total{channel, template_type, status}`
- `notification_processing_seconds{channel}` (histogram)
- `notification_delivery_rate{channel}` (gauge, atualizado a cada hora)

---

## 8. Riscos Top 5

| # | Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|---|
| **R1** | Templates WhatsApp rejeitados pela Meta | Média | Alto | Submeter na semana 1; ter versão alternativa de texto; não depender de templates para lançar email |
| **R2** | Redis indisponível → duplicatas | Baixa | Médio | Fallback: `SELECT 1 FROM notifications WHERE event_id = $1`; Redis com persistência RDB em shared-infra-01 |
| **R3** | WABA suspenso por taxa de bloqueio >2% | Baixa | Crítico | Deduplicação rigorosa; opt-out imediato; monitorar Quality Rating no Meta Business Manager semanalmente |
| **R4** | Delay no processo de aprovação Meta → atraso no launch | Alta | Médio | Submeter templates ANTES de começar o código; lançar email sem WhatsApp se necessário |
| **R5** | Volume explode e excede free tier Resend | Baixa | Baixo | Alerta em 80% do limite; upgrade para $20/mês é automático e cobre 50k emails |

---

## 9. Roadmap — 3 Fases

### Fase 1: Foundation + Email (Semanas 1–2)

**Objetivo:** Alunos recebem confirmação de pagamento por email.

| Sprint | Entregas |
|---|---|
| Sprint 1 (~5 dias) | Setup repositório, migrations (3 tabelas), conexões DB/Redis/RabbitMQ, health endpoints, CI pipeline |
| Sprint 2 (~4 dias) | Resend adapter, template resolver, deduplicação Redis, consumer RabbitMQ, `checkout.confirmed` → email, testes unitários |

**Critério de saída:** Email de recibo enviado automaticamente para todo checkout confirmado, com log de status.

**Ação paralela (semana 1):** Submeter templates WhatsApp para aprovação Meta.

---

### Fase 2: WhatsApp + Observabilidade (Semanas 3–4)

**Objetivo:** Alunos recebem lembretes de aula via WhatsApp.

| Sprint | Entregas |
|---|---|
| Sprint 3 (~4 dias) | Meta Cloud API adapter, webhook receiver, `scheduling.class.reminder` → WhatsApp, retry+backoff, DLQ, testes de integração |
| Sprint 4 (~3 dias) | Prometheus metrics, opt-out handler WhatsApp, `notification_preferences` CRUD, alertas configurados, deploy K3s |

**Critério de saída:** Lembretes de aula via WhatsApp funcionando, opt-out operacional, alertas no Grafana.

---

### Fase 3: Maturidade + Expansão (Semanas 5–8)

**Objetivo:** Plataforma estável, com push notifications e integração com Brio.

| Entrega | Descrição |
|---|---|
| Admin de templates | Interface para atualizar templates sem redeploy |
| Push notifications (Expo) | Adapter para app mobile — depende do app estar em produção |
| Brio escalation (v1.1) | Migrar escaladas do Telegram para notification-service |
| Replay de DLQ | Processo operacional para reprocessar mensagens mortas |
| Dashboard executivo | Relatório mensal de comunicações para o admin |

**Critério de saída:** Delivery rate ≥95% sustentado por 30 dias, DLQ < 5 mensagens/dia, Brio migrado.

---

## 10. Decisões Pendentes

| # | Decisão | Owner | Prazo | Impacto se não resolver |
|---|---|---|---|---|
| **D1** | Submeter templates WhatsApp para aprovação Meta | @devops / @backend | Semana 1 (urgente) | Bloqueia Fase 2 completamente |
| **D2** | Domínio de email do Resend | CEO / @devops | Semana 1 | Bloqueia envio de email |
| **D3** | Número de telefone WABA (WhatsApp Business Account) | CEO | Semana 1 | Bloqueia WhatsApp — pode usar número pessoal ou contratar novo |
| **D4** | Quem é o DPO para LGPD? | CEO | Mês 1 | Risco legal — pode ser o próprio fundador |
| **D5** | Push notifications: esperar app mobile ou MVP sem push? | CEO / @pm | Semana 2 | Define scope da Fase 3 |
| **D6** | Escalada do Brio em v1.1: WhatsApp ou manter Telegram? | CEO | Mês 2 | Define custo e UX das escaladas |
| **D7** | Limite de rate do WhatsApp: Tier 1 (1k/dia) é suficiente? | @data | Mês 1 | Pode precisar de request para upgrade de tier |

---

## Apêndices

### A. Stack Técnica

| Componente | Tecnologia |
|---|---|
| Linguagem | Go + Fiber |
| Porta | :3012 |
| Mensageria | RabbitMQ (consumer) |
| Cache/Dedup | Redis DB7 |
| Banco de dados | PostgreSQL |
| Email | Resend (`github.com/resend/resend-go/v2`) |
| WhatsApp | Meta Cloud API (Graph API v18.0) |
| Push | Expo Push Notifications (P2) |
| Observabilidade | Prometheus + Grafana |
| Deployment | K3s (homelab), 1 replica MVP |
| CI/CD | GitHub Actions + ArgoCD (GitOps) |

### B. Recursos Mínimos K8s (MVP)

```
CPU: 50m request / 200m limit
Memory: 64Mi request / 128Mi limit
```

### C. Cobertura de Testes Mínima

| Pacote | Meta |
|---|---|
| `internal/dedup` | 100% |
| `internal/dispatcher` | 80% |
| `internal/template` | 80% |
| `internal/channel/email` | 70% |
| `internal/channel/whatsapp` | 70% |
| **Total** | **≥70%** |

### D. Casos de Teste P0 (Bloqueiam Deploy)

1. Email enviado após `checkout.confirmed`
2. WhatsApp enviado após `scheduling.class.reminder`
3. Usuário com opt-out não recebe notificação (status: `opted_out`)
4. Falha no provedor resulta em NACK → DLQ (não ACK)
5. Mesmo evento processado duas vezes → apenas 1 mensagem enviada (deduplicação)
6. Após 3 falhas → mensagem vai para DLQ com NACK

---

*PRD gerado pelo @pm a partir das análises de @data, @legal, @backend, @devops, @qa e @techlead em 2026-03-03.*
