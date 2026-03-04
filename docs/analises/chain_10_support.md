# chain_10_support.md — @support
# Playbooks de Suporte: notification-service

**Agente:** @support  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_09_qa.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam os playbooks de suporte:
>
> - **@data:** "Custo de NÃO ter: cliente sem recibo → suporte manual → R$2.500/mês" → cada ticket evitado é ROI direto
> - **@data:** "Taxa de entrega WhatsApp 92-97%" → 3-8% de falhas geram tickets de suporte previsíveis
> - **@data:** "Opt-out rate alerta > 2% WhatsApp" → playbook específico para pico de opt-outs
> - **@data:** "Status `delivered` só atualizado via webhook" → suporte deve entender diferença sent vs. delivered
> - **@finance:** "Fallback: WhatsApp falha → email" → suporte pode verificar se cliente recebeu por canal alternativo
> - **@finance:** "R$2.500/mês em custo de suporte evitado com payment.confirmed" → prioridade máxima para tickets de recibo
> - **@qa:** "Casos críticos: dedup, opt-out imediato, fallback de canal, dead-letter" → cenários mais prováveis de ticket

---

## 1. Categorias de Tickets Esperados

| Categoria | Frequência Estimada | Prioridade |
|-----------|-------------------|------------|
| "Não recebi confirmação de pagamento" | Alta (3-5% das transações) | Alta |
| "Quero parar de receber mensagens" | Média (1-2%/mês) | Alta (legal) |
| "Recebi a mesma mensagem duas vezes" | Baixa (<0,5%) | Média |
| "Recebi mensagem que não me pertence" | Raríssima | Crítica |
| "Não recebi lembrete de aula" | Média | Média |
| "Como altero meu número de WhatsApp?" | Média | Baixa |

---

## 2. Playbook 1 — Cliente Não Recebeu Notificação

### Trigger
> "Não recebi o comprovante de pagamento", "Não recebi o lembrete da minha aula"

### Passos de Diagnóstico

```
PASSO 1: Verificar se o evento foi processado
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Admin Panel: Notifications > Histórico do cliente
→ Filtrar por data do evento e template_type
→ Checar status:

  ┌─────────────────────────────────────────────────────────┐
  │ STATUS ENCONTRADO → AÇÃO                                 │
  ├─────────────────────────────────────────────────────────┤
  │ "sent"     → Notificação saiu do sistema                │
  │              Verificar spam/caixa de entrada (email)     │
  │              Verificar WhatsApp bloqueado               │
  ├─────────────────────────────────────────────────────────┤
  │ "failed"   → Falha no provider                          │
  │              Ver error_code no payload                  │
  │              Reenviar manualmente                       │
  ├─────────────────────────────────────────────────────────┤
  │ "pending"  → Preso na fila                              │
  │              Verificar DLQ no RabbitMQ                  │
  │              Alertar @devops se DLQ > 10 msgs           │
  ├─────────────────────────────────────────────────────────┤
  │ Não encontrado → Evento nunca chegou ao serviço         │
  │              Verificar no checkout-service/scheduling   │
  │              se o evento foi publicado no RabbitMQ      │
  └─────────────────────────────────────────────────────────┘

PASSO 2: Verificar opt-out
━━━━━━━━━━━━━━━━━━━━━━━━━━
→ notification_preferences: opted_in = false?
→ Se sim: cliente havia optado por não receber (informar cliente)
→ Se não: continuar diagnóstico

PASSO 3: Verificar canal alternativo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ @finance: "WhatsApp falha → fallback email"
→ Checar se foi entregue em canal diferente do esperado
→ "Enviamos também por email — verificou?" 

PASSO 4: Reenviar se necessário
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Botão "Reenviar" no admin panel (@frontend)
→ Ou: POST /api/v1/notify com payload original
→ Confirmar com cliente que recebeu
```

### Respostas Padrão ao Cliente

**Status "sent" (mais comum):**
> "Olá [Nome]! Verificamos e a notificação foi enviada com sucesso às [hora] para [email/WhatsApp]. Pode verificar:
> - Email: pasta de spam ou promoções
> - WhatsApp: se nosso número está salvo, mensagens de números não salvos podem ir para 'Desconhecidos'
> Se não encontrar, posso reenviar agora. [Foto do comprovante em PDF também disponível aqui: link]"

**Status "failed":**
> "Olá [Nome]! Identificamos uma falha técnica no envio da sua notificação. Já fizemos o reenvio e você deve receber em instantes. Pedimos desculpas pelo inconveniente."

**Não encontrado (evento não publicado):**
> Escalar para @backend — possível falha no serviço de origem (checkout, scheduling).

---

## 3. Playbook 2 — Pedido de Opt-Out

### Trigger
> "Quero parar de receber mensagens", "Me tira da lista", "SAIR", "STOP"

### ⚠️ Prioridade Legal — Processar IMEDIATAMENTE

```
PASSO 1: Identificar canal solicitado
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ "Quero parar de receber WhatsApp" → opt-out apenas WhatsApp
→ "Não quero mais nenhuma mensagem" → opt-out todos os canais
→ "Remover meu email" → opt-out email

PASSO 2: Processar opt-out
━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Via admin panel: Customer > Notificações > Preferências
→ Desativar canal(is) solicitado(s)
→ Sistema atualiza notification_preferences imediatamente

PASSO 3: Confirmação obrigatória
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Confirmar ao cliente que opt-out foi processado
→ Informar quais notificações CONTINUARÃO (ex: recibos de pagamento por contrato)

PASSO 4: Exceções (LGPD @legal)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Notificações transacionais (payment.confirmed, order.ready) NÃO podem ser desativadas
→ Explicar ao cliente: "Confirmações de pagamento são necessárias para o funcionamento do serviço"
→ Se cliente insistir em não receber NADA: escalar para gerência (cancelamento implícito?)
```

### Resposta Padrão ao Cliente

**Opt-out WhatsApp:**
> "Olá [Nome]! Removemos seu número de WhatsApp das nossas notificações. A partir de agora não receberá mais mensagens neste canal.
>
> Importante: Continuaremos enviando confirmações de pagamento por email, pois são necessárias para comprovar suas transações.
>
> Para reativar quando quiser: [link preferências]"

**Opt-out canal via "SAIR" no WhatsApp:**
> Resposta automática configurada:
> "Seu número foi removido das nossas mensagens automáticas. Para reativar, acesse: [link] ou responda ATIVAR."

---

## 4. Playbook 3 — Notificação Duplicada

### Trigger
> "Recebi a mesma mensagem duas vezes", "Me mandaram o mesmo lembrete 3 vezes"

```
PASSO 1: Confirmar duplicata
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Admin panel: Histórico do cliente, ordenar por template_type + data
→ Verificar se há 2+ registros com mesmo event_id → BUG (dedup falhou)
→ Verificar se são event_ids DIFERENTES → comportamento esperado (2 aulas, 2 pagamentos)

PASSO 2: Classificar causa
━━━━━━━━━━━━━━━━━━━━━━━━━━
Caso A: Mesmo event_id → 2 notificações
  → Bug de deduplicação (Redis falhou ou TTL muito curto)
  → Severidade: ALTA — abrir issue urgente para @backend
  → Verificar Redis DB7: chave notif:{customer_id}:{template}:{event_id} existe?

Caso B: Event_ids diferentes → conteúdo parecido
  → 2 eventos legítimos foram publicados (ex: sistema mandou 2 cobranças)
  → Verificar no serviço de origem (checkout)
  → Possível bug no publisher, não no notification-service

PASSO 3: Compensar o cliente
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Pedido de desculpas + confirmação de que foi corrigido
→ Se causa identificada: informar ETA do fix

PASSO 4: Documentar para @qa
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Registrar: customer_id, event_id, timestamps, canal
→ Acionar @qa para adicionar caso de teste de regressão
```

### Resposta Padrão ao Cliente

> "Olá [Nome]! Pedimos desculpas pela mensagem duplicada. Identificamos o problema e já foi corrigido.
>
> Você não receberá mensagens duplicadas novamente. Caso ocorra novamente, nos informe."

---

## 5. Playbook 4 — Evolution API / WhatsApp Banido (Incidente)

### Trigger
> Alertas de @devops: taxa de entrega WhatsApp caindo abruptamente

```
PASSO 1: Confirmar incidente
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Admin panel: taxa de entrega WhatsApp < 50% por 10 min?
→ Verificar logs Evolution API (error: "number banned" ou "session invalid")

PASSO 2: Comunicar proativamente (@growth: WhatsApp é canal primário)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
→ Postar no canal interno: "[INCIDENTE] WhatsApp indisponível. Notificações críticas sendo redirecionadas para email."

PASSO 3: Ativar fallback
━━━━━━━━━━━━━━━━━━━━━━━━━
→ @backend confirmou fallback automático (WhatsApp falha → email)
→ Verificar que emails estão sendo enviados (Resend dashboard)

PASSO 4: Resolução
━━━━━━━━━━━━━━━━━━━
→ @devops: rotar número ou criar nova instância Evolution
→ Ou: ativar Meta Cloud API (se credentials configuradas)
→ Reprocessar DLQ quando WhatsApp voltar
```

---

## 6. FAQs de Suporte — Respostas Rápidas

| Pergunta | Resposta |
|---------|---------|
| "Como altero meu número de WhatsApp?" | Acessar Settings > Perfil > Telefone; ou admin altera no customer-service |
| "Posso receber só email, sem WhatsApp?" | Sim! Settings > Notificações > desativar WhatsApp |
| "Por que recebi mensagem tarde da noite?" | Sistema usa horário preferencial. Se incomodou, configurar janela em Settings |
| "Meu email estava errado, perdi o recibo" | Admin pode reenviar via admin panel (botão Reenviar) |
| "Recebi notificação de outra pessoa" | Escalação imediata para @backend (bug de segurança) |

---

## 7. Métricas de Suporte a Monitorar

Derivadas de @data (campo `status` da tabela notifications):

| Métrica | Fórmula | Meta |
|---------|---------|------|
| Tickets de "não recebi" | tickets_nao_recebi / total_notificacoes_enviadas | < 1% |
| Tempo de resolução | avg(resolved_at - opened_at) | < 2h |
| Taxa de reenvio manual | reenvios_manuais / total_notificacoes | < 2% |
| Opt-outs via suporte | opt_outs_via_suporte / total_opt_outs | < 10% (maioria deve ser self-service) |

---

## 8. Escalação

| Situação | Escalar Para |
|---------|-------------|
| Status "pending" > 1h | @devops (DLQ ou consumer parado) |
| Dedup falhou (2 notifs mesmo event_id) | @backend + @qa |
| WhatsApp ban | @devops (novo número) + @legal (risco Meta ToS) |
| Cliente recebeu dados de outro cliente | @backend URGENTE (data leak) |
| Opt-out não processado após 1h | @backend URGENTE (LGPD) |
