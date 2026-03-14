# chain_02_finance.md — @finance
# Análise Financeira: notification-service

**Agente:** @finance  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md

---

## Alinhamento com @data

> Citações diretas de chain_01_data.md que guiam as decisões financeiras:
>
> - **"Canal mais barato: Push (Expo) = gratuito. Email via Resend = gratuito até 3k/mês"** → confirma que o custo marginal dos primeiros meses é quase zero se WhatsApp for via Evolution API self-hosted
> - **"WhatsApp via Meta Cloud API = $0,06/conversa"** → custo variável que escala com volume; monitorar ativamente
> - **"`trial.expiring` tem maior potencial de conversão (15-25%)"** → maior ROI por notificação; priorizar delivery
> - **"Volume base M3: ~5.000 notificações/mês com 10 tenants"** → projeção base para cálculo de custo
> - **"Fallback para email (gratuito) reduz custo de retry"** → estratégia de contenção de custo

---

## 1. Estrutura de Custos por Canal

### 1.1 Email — Resend
| Tier | Volume | Custo |
|------|--------|-------|
| Free | até 3.000/mês | $0 |
| Pro | até 50.000/mês | $20/mês |
| Business | até 100.000/mês | $89/mês |

**Estratégia IIT:** Permanecer no tier gratuito até M6. Transacional tem boa reputação → baixo risco de spam.

### 1.2 WhatsApp — Opções

#### Opção A: Evolution API (self-hosted) — **RECOMENDADA para MVP**
- Custo: incluso na infra K3s (já existente)
- Limitação: 1 número por instância, risco de ban em uso comercial intensivo
- Adequado para: volume < 1.000 msg/dia por conta

#### Opção B: Meta Cloud API
- Custo por conversa: **$0,06 USD** (conversa = janela de 24h)
- Volume incluso gratuito: 1.000 conversas/mês
- Custo acima de 1.000: $0,06/conversa
- Vantagem: escala, confiabilidade, templates oficiais
- Bloqueador: aprovação de templates (1-7 dias úteis)

#### Comparativo de custo WhatsApp
| Cenário | Evolution API | Meta Cloud API |
|---------|---------------|----------------|
| 500 conversas/mês | ~$5 infra | $0 (incluso) |
| 2.000 conversas/mês | ~$5 infra | $60 |
| 10.000 conversas/mês | ~$20 infra* | $540 |
| 50.000 conversas/mês | ~$80 infra* | $2.940 |

*Estimativa de custo de infra adicional (CPU/RAM para múltiplas instâncias Evolution)

**Decisão financeira:** Evolution API até atingir 2.000 conversas/mês. Migrar para Meta Cloud API a partir de M6-M9 quando volume justificar confiabilidade enterprise.

### 1.3 Push — Expo Push API
- Custo: **$0** (gratuito ilimitado para push notifications)
- Dependência: app mobile instalado + opt-in do usuário
- SLA: melhor esforço (sem garantia de entrega)

---

## 2. ROI por Tipo de Notificação

### 2.1 `class.reminder` — Lembrete de Aula (WhatsApp)
**Impacto:** Reduz no-show em 60% (dado de @data: benchmark de mercado)

Exemplo para academia com 50 alunos, 3 aulas/semana:
- Sem lembrete: 25% no-show = 37,5 vagas desperdiçadas/semana
- Com lembrete: 10% no-show = 15 vagas desperdiçadas/semana
- Vagas recuperadas: 22,5/semana = ~90/mês
- Se academia cobra R$150/aula avulsa: **R$13.500/mês de capacidade recuperada**
- Custo do lembrete: 50 WhatsApps/semana = 200/mês ≈ $12 USD

**ROI da notificação:** > 1.000x

### 2.2 `payment.confirmed` — Recibo de Pagamento (Email + WhatsApp)
**Impacto:** Elimina tickets de suporte "não recebi comprovante"

Estimativa:
- Sem notificação: 10% dos pagadores abrem ticket = 1h suporte @ R$50/h
- Com 500 pagamentos/mês: 50 tickets × R$50 = **R$2.500/mês em suporte evitado**
- Custo: gratuito (email) + $30 WhatsApp = $30/mês

**ROI:** > 80x

### 2.3 `trial.expiring` — Trial Expirando (WhatsApp + Email)
**Impacto:** 15-25% convertem para plano pago (dado de @data)

Exemplo com 100 trials expirando/mês:
- Sem notificação: 5% convertem = 5 novos pagantes
- Com notificação: 20% convertem = 20 novos pagantes
- Delta: +15 clientes × ARPU R$100 = **R$1.500 MRR adicional**
- Custo: $6 USD/mês

**ROI:** Incalculável em termos de LTV — esta é a notificação de maior impacto financeiro.

### 2.4 `payment.overdue` — Inadimplência (WhatsApp)
**Impacto:** 30-50% regularizam em 24h após notificação

Exemplo: R$30.000 MRR com 5% inadimplência = R$1.500 em risco
- Com notificação: 40% regularizam = **R$600 MRR recuperado**
- Custo: $3/mês

**ROI:** 200x

---

## 3. Custo de NÃO ter o Notification Service

| Situação | Consequência | Custo Estimado/mês |
|----------|-------------|-------------------|
| Sem recibo de pagamento | Clientes abrem ticket, revisão manual | R$2.500 (suporte) |
| Sem lembrete de aula | No-show alto, insatisfação, churn | R$500 (churn ~2 alunos/mês) |
| Sem aviso trial expirando | Perda de conversão | R$1.500 MRR/mês |
| Sem aviso inadimplência | Churn forçado, cobrança manual | R$800 (suporte + perdas) |
| **Total** | | **~R$5.300/mês** |

**Custo total do serviço (M3):** ~$35/mês (~R$175/mês)  
**Economia gerada:** ~R$5.300/mês  
**Payback:** Imediato — ROI positivo desde o primeiro mês

---

## 4. Projeção de Volume × Custo por Canal

### Premissas de crescimento:
- M3: 10 tenants, ~500 notificações/dia
- M6: 25 tenants, ~1.500 notificações/dia
- M12: 75 tenants, ~5.000 notificações/dia

### Custo mensal projetado (USD):

| Canal | M3 (15k notif) | M6 (45k notif) | M12 (150k notif) |
|-------|----------------|----------------|-----------------|
| Email (Resend) | $0 (free tier) | $20 (Pro) | $89 (Business) |
| WhatsApp Evolution* | $5 infra | $15 infra | $50 infra |
| WhatsApp Meta Cloud** | $0 (1k free) | $120 | $500 |
| Push (Expo) | $0 | $0 | $0 |
| **Total (Evolution)** | **~$5** | **~$35** | **~$139** |
| **Total (Meta Cloud)** | **~$0** | **~$140** | **~$589** |

*Assumindo 30% das notificações via WhatsApp  
**Meta Cloud apenas se migrar do Evolution

### Recomendação @finance:
- **M3:** Evolution API + Resend free = custo marginal ~$5/mês
- **M6:** Avaliar migração WhatsApp → Meta Cloud se volume ultrapassar 2k conversas/mês
- **M12:** Budget de $200/mês para infra de notificações é conservador e sustentável

---

## 5. Budget e Controle de Gastos

### Alertas de custo (implementar desde M1):
- WhatsApp conversations > 800/mês → alertar (próximo do limite gratuito Meta Cloud)
- Resend emails > 2.500/mês → alertar (próximo do free tier)
- Custo total > $50/mês → review financeiro

### Otimizações de custo:
1. **Fallback strategy**: WhatsApp falha → tentar email (gratuito)
2. **Deduplicação rigorosa**: evitar reenvio = evitar custo duplo
3. **Batch de reminders**: consolidar múltiplos lembretes em 1 mensagem quando possível
4. **Canal preference**: se cliente preferir email, não gastar WhatsApp

---

## 6. Métricas Financeiras a Monitorar

Derivadas das métricas de @data (`delivered_at`, `opened_at`, `clicked_at`):

| Métrica | Como Medir | Frequência |
|---------|-----------|------------|
| Custo por notificação entregue | custo_canal / delivered_count | Semanal |
| Revenue influenciado por canal | conversões atribuídas × ARPU | Mensal |
| Custo de retry (falhas) | attempt_count > 1 × custo_canal | Semanal |
| ROI por template_type | revenue_influenciado / custo_envio | Mensal |

---

## 7. O Que @finance Recomenda aos Demais Agentes

- **@backend:** Implementar tracking de `attempt_count` — cada retry tem custo
- **@devops:** Monitorar evolução API com múltiplas contas se volume crescer
- **@growth:** WhatsApp tem melhor ROI mas maior custo — usar com inteligência
- **@techlead:** Decisão de canal primário deve considerar custo total de ownership (TCO)
