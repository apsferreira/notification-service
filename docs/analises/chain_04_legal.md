# chain_04_legal.md — @legal
# Análise Legal: notification-service

**Agente:** @legal  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam a análise legal:
>
> - **@data:** "`notification_preferences` é a fonte de verdade de consentimento (LGPD)" → confirma necessidade de tabela dedicada com `opted_in_at` e `opt_in_source`
> - **@data:** "Opt-out deve ser processado em < 1 hora (meta: imediato)" → requisito de implementação com obrigação legal
> - **@data:** "`opted_out_at` e `opt_out_source`" → campos obrigatórios por LGPD para rastreabilidade
> - **@finance:** "Meta Cloud API: aprovação de templates 1-7 dias úteis — bloqueador" → impacto no cronograma de lançamento
> - **@finance:** "Evolution API: risco de ban em uso comercial intensivo" → risco legal além do técnico

---

## 1. LGPD — Requisitos Mandatórios

### 1.1 Base Legal para Notificações

O notification-service opera sob **duas bases legais** distintas (Art. 7º LGPD):

| Tipo de Notificação | Base Legal | Justificativa |
|--------------------|-----------|--------------|
| `payment.confirmed` | **Execução de contrato** (Art. 7º, V) | Necessário para execução do serviço pago |
| `class.reminder` | **Legítimo interesse** (Art. 7º, IX) ou **consentimento** | Lembrete benéfico ao titular |
| `order.ready` | **Execução de contrato** (Art. 7º, V) | Parte da prestação do serviço |
| `trial.expiring` | **Legítimo interesse** (Art. 7º, IX) | Informação relevante ao titular |
| `payment.overdue` | **Execução de contrato** + **obrigação legal** | Cobrança de dívida legítima |
| Mensagens de marketing/upsell | **Consentimento explícito** (Art. 7º, I) | OBRIGATÓRIO opt-in |

**⚠️ Regra crítica:** Notificações transacionais (pagamento, pedido) podem ser enviadas sem opt-in explícito. Mensagens comerciais/marketing EXIGEM consentimento.

### 1.2 Requisitos de Consentimento (Art. 8º LGPD)

O consentimento para notificações comerciais deve ser:
- **Livre:** sem coerção ou condição de uso do serviço
- **Informado:** cliente sabe o que vai receber e com qual frequência
- **Inequívoco:** não pode ser presumido — deve ser ação afirmativa (checkbox, botão)
- **Granular por canal:** opt-in para WhatsApp ≠ opt-in para email

### 1.3 Schema LGPD-Compliant

A tabela `notification_preferences` de @data já está corretamente estruturada. Adições obrigatórias:

```sql
ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS
    consent_text TEXT,              -- texto exato apresentado ao usuário no opt-in
    consent_version VARCHAR(20),    -- versão da política (ex: "2026-01")
    ip_address INET,                -- IP no momento do opt-in (evidência)
    user_agent TEXT;                -- browser/app no momento do opt-in

-- Log de auditoria imutável (INSERT ONLY — nunca UPDATE/DELETE)
CREATE TABLE consent_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id     UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    channel         VARCHAR(20) NOT NULL,
    action          VARCHAR(20) NOT NULL,  -- opted_in | opted_out | consent_updated
    consent_text    TEXT,
    consent_version VARCHAR(20),
    ip_address      INET,
    user_agent      TEXT,
    performed_at    TIMESTAMPTZ DEFAULT NOW(),
    performed_by    VARCHAR(50)            -- 'customer' | 'admin' | 'system'
);
```

### 1.4 Opt-Out — Requisitos Legais

**Lei:** Opt-out deve ser honrado imediatamente (LGPD Art. 18, VI — "oposição").

**Implementação obrigatória:**
1. Link de opt-out em TODOS os emails (requisito CAN-SPAM e LGPD)
2. Resposta "SAIR" ou "STOP" no WhatsApp deve processar opt-out imediatamente
3. Endpoint `POST /api/v1/notifications/opt-out` acessível sem autenticação (via token único)
4. Confirmação de opt-out enviada ao cliente após processamento
5. Opt-out de um canal NÃO cancela outros canais (granularidade)

**Prazo de processamento:** Imediato (técnico) ou em até 15 dias (legal). Meta: < 1 hora.

---

## 2. Meta Cloud API — Bloqueadores Legais e Operacionais

### 2.1 Aprovação de Templates (BLOQUEADOR CRÍTICO)

Templates do WhatsApp Business API precisam de aprovação prévia da Meta:
- **Prazo:** 1-7 dias úteis por template
- **Categorias permitidas:**
  - `UTILITY`: confirmações, atualizações de conta → aprovação mais rápida
  - `MARKETING`: ofertas, promoções → aprovação mais rigorosa + custo diferente
  - `AUTHENTICATION`: OTPs → não aplicável ao notification-service

**Templates mínimos para MVP (criar e submeter antes do go-live):**

| Template Name | Categoria | Variáveis |
|--------------|-----------|-----------|
| `payment_confirmed` | UTILITY | {{nome}}, {{valor}}, {{data}} |
| `class_reminder` | UTILITY | {{nome}}, {{aula}}, {{horario}}, {{data}} |
| `order_ready` | UTILITY | {{nome}}, {{restaurante}} |
| `trial_expiring` | UTILITY | {{nome}}, {{dias_restantes}}, {{link}} |
| `payment_overdue` | UTILITY | {{nome}}, {{valor}}, {{link_pagamento}} |
| `subscription_paused` | UTILITY | {{nome}}, {{data_reativacao}} |

**⚠️ Impacto no cronograma:** Submeter templates com **pelo menos 10 dias de antecedência** ao go-live.

### 2.2 Política de Uso da Meta

Proibições que impactam o planejamento de @growth:
- ❌ Templates de marketing não podem ser enviados para usuários que não iniciaram conversa nas últimas 24h (janela de serviço)
- ❌ Limites de rate por número (máx. 1.000 conversas/dia em tier inicial)
- ❌ Número de telefone deve ser verificado (Business Verification)
- ✅ Templates UTILITY podem ser enviados a qualquer momento com opt-in

### 2.3 Evolution API — Riscos Legais

- **Risco:** não é API oficial da Meta → ToS da WhatsApp proíbe automação não oficial
- **Consequência:** ban do número (perda do canal sem aviso prévio)
- **Recomendação legal:** Evolution API apenas para MVP/testes. Migrar para Meta Cloud API antes de uso em produção com dados de clientes reais.

---

## 3. Lei do Spam (Anti-Spam) e CAN-SPAM

### 3.1 Requisitos para Email Comercial (Lei Brasileira + Boas Práticas)

Todo email comercial deve conter:
- [ ] Identificação clara do remetente (nome do tenant/empresa)
- [ ] Endereço físico do remetente
- [ ] Link de descadastro (unsubscribe) **funcionando e visível**
- [ ] Assunto do email não pode ser enganoso
- [ ] Marcação "publicidade" ou "propaganda" se conteúdo for comercial

**Configuração Resend obrigatória:**
- SPF, DKIM e DMARC configurados no domínio
- Bounce handling: emails com hard bounce → marcar opted_out automaticamente
- Spam complaints: integrar webhook da Resend → opt-out imediato

### 3.2 WhatsApp Anti-Spam

A Meta monitora:
- Taxa de bloqueio pelo usuário > 2% → número suspenso
- Mensagens sem template aprovado para iniciar conversa → suspensão
- Volume anormal de envios → revisão automática

**Monitorar via dados de @data:** opt-out rate por canal. Se WhatsApp > 2% → parar campanha e revisar.

---

## 4. Retenção de Dados

| Dado | Retenção | Fundamento |
|------|----------|------------|
| `notifications` (tabela principal) | 2 anos | Prazo prescricional de disputas |
| `consent_audit_log` | 5 anos | LGPD Art. 15 + prazo prescricional |
| `delivery_events` | 1 ano | Operacional |
| Dados de opt-out | Permanente | Evidência de respeito ao direito |

---

## 5. Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Templates Meta rejeitados | Alta (1ª vez) | Alto | Enviar com 10d de antecedência; ter fallback email |
| Número Evolution banido | Média | Alto | Migrar para Meta Cloud API antes de escalar |
| Opt-out não processado | Baixa (se implementado) | Alto (LGPD multa) | Teste automatizado de opt-out |
| Envio para menor sem consentimento dos pais | Baixa | Alto | Validar idade no customer-service |
| Breach de dados de contato | Baixa | Alto | Dados de contato não trafegar em logs |

---

## 6. Checklist Legal para Go-Live

- [ ] `notification_preferences` implementada com todos os campos LGPD
- [ ] `consent_audit_log` implementada e imutável
- [ ] Endpoint de opt-out sem autenticação funcionando
- [ ] Link de unsubscribe em todos os emails
- [ ] Templates Meta submetidos e aprovados (mín. 10d antes)
- [ ] Webhook de spam complaint (Resend) → opt-out automático
- [ ] Política de privacidade atualizada mencionando canais de notificação
- [ ] Teste de opt-out: enviar → opt-out → confirmar que próxima mensagem não chega
- [ ] Retenção de dados configurada (pg_partman ou cron de purge)
