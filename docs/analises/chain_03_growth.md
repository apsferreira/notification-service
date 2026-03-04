# chain_03_growth.md — @growth
# Análise de Growth: notification-service

**Agente:** @growth  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam a estratégia de growth:
>
> - **@data:** "WhatsApp tem open rate 80% vs email 35% — priorizar WhatsApp para ações de conversão"
> - **@data:** "`trial.expiring` tem potencial de conversão 15-25%" → oportunidade de upsell
> - **@data:** "`clicked_at` permite A/B test de CTAs" → base para experimentos de growth
> - **@finance:** "ROI de `trial.expiring` → +15 clientes × ARPU R$100 = R$1.500 MRR adicional" → maior alavanca de receita
> - **@finance:** "WhatsApp tem melhor ROI mas maior custo — usar com inteligência" → priorizar conversão, não volume
> - **@finance:** "Evolution API até 2.000 conversas/mês" → janela de testes gratuita para experimentos

---

## 1. Notificações como Canal de Crescimento

O notification-service não é apenas operacional — é um canal de marketing de alta performance:
- **Permissioned:** cliente já optou em receber
- **Contextual:** enviado no momento de maior relevância (pós-pagamento, pré-aula)
- **Measurable:** `delivered_at`, `opened_at`, `clicked_at` permitem atribuição precisa
- **Cheap:** custo marginal próximo de zero vs. custo de aquisição via ads

### Regra de Ouro de Growth:
> **Usar notificação transacional como gatilho de crescimento — nunca como spam.**

---

## 2. Playbook de Upsell Pós-Pagamento

### 2.1 Sequência pós `payment.confirmed`
```
T+0s:   Email — Recibo de pagamento (obrigatório, transacional)
T+0s:   WhatsApp — "Seu pagamento foi confirmado! 🎉"
T+1h:   WhatsApp — Upsell contextual (apenas se plano básico)
         "Sabia que o plano Premium inclui aulas ilimitadas? Upgrade por +R$50/mês"
T+7d:   Email — "Como estão seus primeiros dias?" + NPS rápido
```

### 2.2 Regras de upsell (não virar spam):
- Apenas 1 mensagem de upsell por evento de pagamento
- Não enviar se cliente já tem plano premium
- Não enviar se cliente fez opt-out de mensagens comerciais
- Respeitar `opted_in` da `notification_preferences` (base @data/@legal)

---

## 3. WhatsApp como Canal Primário de Conversão

Com base no open rate de 80% identificado por @data, WhatsApp deve ser o canal primário para:

| Use Case | Canal | CTA |
|----------|-------|-----|
| Trial expirando | WhatsApp + Email | "Assinar agora" com link direto |
| Upsell pós-pagamento | WhatsApp | "Ver plano Premium" |
| Reativação de churn | WhatsApp | "Voltar com 20% desconto" |
| Lembrete de aula | WhatsApp | "Confirmar presença" |
| Inadimplência | WhatsApp | "Regularizar agora" (link Asaas) |

### Reativação de Churn — Sequência
```
Trigger: subscription.cancelled (evento checkout)
T+0:   WhatsApp "Sentimos sua falta [nome]! Quer nos contar por quê cancelou?"
T+3d:  WhatsApp "Que tal voltar com 1 mês grátis?" (se não respondeu)
T+7d:  Email "Oferta exclusiva: 20% off no primeiro mês de volta"
T+14d: PARAR (não mandar mais — respeitar decisão)
```

> **@finance validou:** reativar 1 churned customer via notificação custa ~$0,12 vs. CAC de aquisição de R$50-200.

---

## 4. Estratégia de Opt-In para Maximizar Alcance

### 4.1 Momento do Opt-In (LGPD compliant)

**Regra:** opt-in explícito obrigatório. Mas o MOMENTO do opt-in importa para conversão.

| Momento | Conversão Esperada | Canal |
|---------|-------------------|-------|
| Durante signup (cold) | 40-60% | Email obrigatório |
| Pós-pagamento (warm) | 70-85% | WhatsApp opt-in |
| Check-in na academia (hot) | 80-90% | Push notification opt-in |

**Estratégia:** não pedir todos os opt-ins no signup. Pedir no momento mais relevante.

### 4.2 Fluxo de Opt-In WhatsApp
```
1. Cliente finaliza primeiro pagamento
2. Tela: "Quer receber lembretes de aula e confirmações pelo WhatsApp?"
   [Sim, quero!] [Agora não]
3. Se sim: gravar opted_in=true, opted_in_at=NOW(), opt_in_source='post_payment'
4. Primeira mensagem: "Oi [nome]! Vou te avisar sobre suas aulas e pagamentos. 
   Para parar de receber, responda SAIR."
```

### 4.3 Double Opt-In para WhatsApp (recomendado)
```
1. Admin cadastra número do cliente
2. System envia: "Olá! Para receber atualizações da [Academia], responda SIM"
3. Cliente responde SIM → opted_in = true (evidência de consentimento)
4. Se não responder em 48h → não enviar notificações WhatsApp
```

> Double opt-in protege de números incorretos e satisfaz LGPD (@legal confirmará).

---

## 5. A/B Testing com Dados de @data

Usando `clicked_at` e `opened_at` de chain_01_data.md:

### Experimentos prioritários:
1. **CTA de trial expirando:**
   - Variante A: "Assinar agora por R$99/mês"
   - Variante B: "Continuar por apenas R$99/mês 🎯"
   - Métrica: `clicked_at` rate

2. **Timing do lembrete de aula:**
   - 24h antes vs. 2h antes
   - Métrica: checkin_rate (aula confirmada)

3. **Canal para inadimplência:**
   - WhatsApp vs. Email vs. ambos
   - Métrica: recovery_rate em 24h

### Implementação mínima:
- Campo `ab_variant` no payload de notificação
- Agrupar por variant em analytics
- 2 semanas por experimento, mínimo 100 eventos por variante

---

## 6. Métricas de Growth a Rastrear

| Métrica | Fórmula | Meta M3 |
|---------|---------|---------|
| Trial → Paid (notificação influenciada) | conversões_com_notif / total_trials | 20% |
| Churn Reactivation Rate | reativados_30d / churned_30d | 10% |
| WhatsApp Opt-In Rate | opted_in_whatsapp / total_customers | 70% |
| Revenue Influenced by Notification | GMV de clientes que clicaram em notif | tracking |
| NPS via notificação | respostas NPS / enviados | 15% response |

---

## 7. Calendário de Ações Growth (M1-M3)

| Semana | Ação |
|--------|------|
| W1 | Implementar opt-in WhatsApp no fluxo de pagamento |
| W2 | Ativar sequência `trial.expiring` |
| W3 | Ativar upsell pós-`payment.confirmed` |
| W4 | Primeiro A/B test (CTA trial) |
| W6 | Ativar sequência de reativação de churn |
| W8 | Review de métricas + ajuste de timing |
| W10 | Expandir para NPS via WhatsApp |
| W12 | Relatório M3: ROI total do canal notificação |
