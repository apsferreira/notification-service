# Analysis Data — notification-service

**Agente:** @data  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. Benchmarks de Deliverability — Email (Brasil)

### Taxas médias de mercado (B2C transacional, 2024-2025)

| Métrica | Referência Mercado Brasil | Meta IIT |
|--------|--------------------------|----------|
| Delivery rate (email transacional) | 97–99% | ≥98% |
| Open rate (transacional) | 45–65% | ≥50% |
| Open rate (marketing) | 18–28% | N/A (fora do escopo) |
| Bounce rate (hard) | <0,5% | <0,3% |
| Bounce rate (soft) | <3% | <2% |
| Spam complaint rate | <0,08% (limite Resend/Google) | <0,05% |

### Fatores críticos de deliverability no Brasil

1. **Autenticação SPF + DKIM + DMARC** — obrigatório para evitar spam
2. **Domínio de envio dedicado** — usar `noreply@institutoitinerante.com.br`
3. **Reputação de IP/domínio** — Resend gerencia warmup
4. **Horário de envio** — transacionais devem ser imediatos (< 60s após evento)
5. **Unsubscribe em 1 clique** — obrigatório pelo Google/Yahoo (2024)

### Resend (provedor escolhido)

- SLA de entrega: 99,9% uptime
- Latência de envio: p50 < 2s, p99 < 10s
- Webhooks: `email.delivered`, `email.bounced`, `email.complained`
- Free tier: 3.000 emails/mês, 100/dia — suficiente para MVP
- Paid: $20/mês para 50k emails
- SDK Go: `github.com/resend/resend-go/v2`

---

## 2. Benchmarks de Deliverability — WhatsApp (Meta Cloud API, Brasil)

Brasil é o maior mercado WhatsApp do mundo (~147M usuários ativos).

| Métrica | WhatsApp Transacional (BR) | Meta IIT |
|--------|--------------------------|---------|
| Delivery rate | 95–98% | ≥95% |
| Open rate | 85–95% | ≥85% |
| Read rate em 5 min | 70–80% | N/A |

### Meta Cloud API — Limites e Regras

- **Template Messages (HSM):** obrigatório para primeiro contato ou após 24h sem interação. Templates precisam de aprovação prévia da Meta (1-7 dias úteis).
- **Limites de rate por WABA:**
  - Tier 1 (padrão): 1.000 conversações/dia
  - Tier 2: 10.000/dia
  - Tier 3: 100.000/dia
- **Custo por conversa (Brasil):** ~$0,0625 USD por conversa de utilidade
- **Webhook de status:** `sent`, `delivered`, `read`, `failed`

### Templates Transacionais Recomendados (jiu-jitsu-academy)

```
TIPO: class_reminder | CATEGORIA: UTILITY
TEXTO: "Olá {{1}}, sua aula de jiu-jitsu está confirmada para amanhã às {{2}}.
Endereço: {{3}}. Para cancelar, responda CANCELAR."

TIPO: payment_confirmed | CATEGORIA: UTILITY
TEXTO: "Pagamento de R$ {{1}} confirmado para {{2}}. Obrigado, {{3}}!"

TIPO: subscription_expiring | CATEGORIA: UTILITY
TEXTO: "Olá {{1}}, sua mensalidade vence em {{2}} dias. Renove em: {{3}}"
```

---

## 3. Melhores Práticas de Templates

1. **Personalização imediata** — nome do usuário logo no início (+15% open rate)
2. **Uma ação por mensagem** — não misturar informações
3. **CTA claro e único** — instrução direta
4. **Urgência contextual** — "amanhã às 19h" em vez de "em breve"
5. **Linguagem conversacional** — WhatsApp é informal; email pode ser mais formal

### Schema notification_templates (REQ-NT-01)

```sql
notification_templates (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100),      -- 'class.reminder', 'checkout.confirmed', etc.
    channel VARCHAR(20),          -- 'email', 'whatsapp', 'push'
    subject VARCHAR(200),         -- apenas email
    body_template TEXT,           -- com placeholders {{variavel}}
    variables JSONB,              -- lista de variáveis esperadas
    active BOOLEAN DEFAULT true,
    meta_template_id VARCHAR(100),-- ID aprovado pela Meta
    created_at TIMESTAMP DEFAULT NOW()
)
```

---

## 4. Recomendações Finais

1. Templates WhatsApp pré-aprovados antes do launch — processo leva 1-7 dias úteis
2. Não usar email para lembretes de aula — WhatsApp tem 10x mais abertura
3. Monitorar delivery rate por canal — alerta se `delivery_rate < 90%` em janela de 1h
4. Volume inicial estimado: ~500 notificações/mês no MVP → free tier Resend suficiente
5. Variáveis de ambiente necessárias:
   - `RESEND_API_KEY`, `RESEND_FROM_EMAIL`
   - `META_WA_PHONE_NUMBER_ID`, `META_WA_ACCESS_TOKEN`, `META_WA_API_VERSION`
