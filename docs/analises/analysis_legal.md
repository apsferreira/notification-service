# Analysis Legal — notification-service

**Agente:** @legal  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. LGPD para Notificações (Lei 13.709/2018)

### Base Legal para Envio de Notificações

O envio de notificações transacionais no contexto do IIT pode se apoiar em diferentes bases legais da LGPD:

| Tipo de Notificação | Base Legal (Art. 7º LGPD) | Observações |
|---------------------|--------------------------|-------------|
| Confirmação de pagamento | Execução de contrato (inc. V) | Necessária para cumprimento do serviço |
| Lembrete de aula | Legítimo interesse (inc. IX) | Benefício direto ao titular |
| Vencimento de mensalidade | Execução de contrato (inc. V) | Obrigação contratual |
| Promoções/marketing | Consentimento (inc. I) | Requer opt-in explícito |

**Atenção:** O notification-service do IIT opera exclusivamente com notificações transacionais — não envia marketing. Isso simplifica a gestão legal.

### Opt-Out Obrigatório

Mesmo notificações transacionais precisam de mecanismo de cancelamento (Art. 18, LGPD):

1. **Email:** Unsubscribe no rodapé de TODOS os emails. Link válido por ≥30 dias.
2. **WhatsApp:** Instrução clara para responder "PARAR" ou "CANCELAR". Ao receber, a plataforma deve marcar `opt_out = true` na tabela de preferências e cessar envios.
3. **Push:** Configuração no app; usuário pode desativar via SO.

### Schema de Consentimento/Preferências

```sql
notification_preferences (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    channel VARCHAR(20),        -- 'email', 'whatsapp', 'push'
    event_category VARCHAR(50), -- 'transactional', 'reminder', 'marketing'
    opted_in BOOLEAN DEFAULT true,
    opted_out_at TIMESTAMP,
    opt_out_reason VARCHAR(200),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(customer_id, channel, event_category)
)
```

### Retenção de Dados

- Logs de notificações (`notifications` table): reter por **24 meses** (padrão para resolução de disputas)
- Dados de opt-out: reter indefinidamente (prova de conformidade)
- Conteúdo de mensagens com dados pessoais: **não logar payload completo em plaintext**; usar referência (`event_id`) ou mascarar

---

## 2. Meta Cloud API — Termos de Serviço Relevantes

### Restrições Críticas (WhatsApp Business Policy)

1. **Proibição de spam:** Meta proíbe explicitamente envio de mensagens em massa não solicitadas. Violação → suspensão da conta WABA.
2. **Templates obrigatórios:** Mensagens fora de janela de 24h DEVEM usar templates aprovados. Enviar mensagens livres fora da janela = violação.
3. **Opt-in explícito requerido:** Meta exige que o usuário tenha dado opt-in para receber mensagens do business. Recomendado: coletar durante cadastro com checkbox explícito.
4. **Conteúdo proibido em templates:**
   - Conteúdo de cobrança agressiva
   - Links encurtados (preferir domínio próprio)
   - Múltiplos CTAs conflitantes
5. **Política de qualidade:** Meta monitora rate de bloqueio de usuários. Se >2% dos destinatários bloquearem o número, o WABA vai para revisão.

### Processo de Aprovação de Templates

1. Criar template no Meta Business Manager
2. Aguardar aprovação (1-7 dias úteis, geralmente 24h)
3. Status: `PENDING` → `APPROVED` / `REJECTED`
4. Templates rejeitados: revisar e resubmeter (não há limite de tentativas)
5. Templates aprovados podem ser pausados pela Meta se gerarem reclamações

### Gestão de Opt-In para WhatsApp (IIT)

Fluxo recomendado para jiu-jitsu-academy:
```
Cadastro do aluno
    → Checkbox: "Aceito receber lembretes de aula e comunicados via WhatsApp"
    → Registrar: opted_in=true, channel='whatsapp', timestamp, IP
    → Armazenar telefone com DDI (+55)
```

---

## 3. Proibição de Spam — Regulação Brasileira

### Marco Legal Aplicável

- **LGPD Art. 18:** Direito de oposição ao tratamento de dados (inclui notificações)
- **CDC Art. 43:** Direito de cancelar cadastro em listas de divulgação
- **Código de Autorregulamentação para Prática de E-mail Marketing (ABEMD):** referência de mercado, não lei, mas orienta práticas

### Boas Práticas Anti-Spam

1. **Não comprar listas** — somente enviar para usuários que se cadastraram organicamente
2. **Honrar opt-out em até 10 dias** (lei sugere imediatamente; implementar em tempo real)
3. **Não enviar mesma mensagem múltiplas vezes** — deduplicação Redis (REQ-NT-02) atende isso
4. **From e Reply-To claros** — identificar o Instituto Itinerante claramente
5. **Frequência razoável** — para jiu-jitsu-academy: máximo 1 lembrete por aula, não múltiplos

---

## 4. Gestão de Consentimento — Implementação

### Fluxo de Opt-Out Automático (WhatsApp)

```go
// Webhook handler para mensagens recebidas
func handleIncomingWhatsApp(msg WebhookMessage) {
    text := strings.ToUpper(strings.TrimSpace(msg.Text))
    if text == "PARAR" || text == "CANCELAR" || text == "STOP" {
        repo.SetOptOut(ctx, msg.From, "whatsapp")
        sendTemplateMessage(msg.From, "opt_out_confirmation_template")
    }
}
```

### Verificação Antes do Envio

```go
func (s *NotificationService) Send(ctx context.Context, n Notification) error {
    pref, err := s.prefRepo.Get(ctx, n.CustomerID, n.Channel)
    if err == nil && !pref.OptedIn {
        return ErrOptedOut // não enviar, não é falha, não retentar
    }
    // prosseguir com envio
}
```

---

## 5. Checklist de Conformidade LGPD

- [ ] Política de Privacidade publicada em `institutoitinerante.com.br/privacidade`
- [ ] Mecanismo de opt-out funcional para cada canal
- [ ] Logs de envio sem dados pessoais em plaintext
- [ ] Registro de base legal para cada tipo de notificação
- [ ] Processo de resposta a requisições de titulares (acesso, exclusão) — prazo legal: 15 dias
- [ ] DPO nomeado ou responsável identificado (para organizações de pequeno porte, pode ser o próprio fundador)
