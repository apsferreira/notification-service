# chain_08_ai.md — @ai
# Inteligência Artificial: notification-service

**Agente:** @ai  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_05_backend.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam as decisões de IA:
>
> - **@data:** "`opened_at` por canal e por horário → permite identificar padrão de engajamento por cliente" → base para timing inteligente
> - **@data:** "Taxa de abertura WhatsApp 80% vs Email 35%" → IA deve confirmar se varia por horário/perfil
> - **@data:** "`clicked_at` permite A/B test de CTAs" → dados para treinar preferências de conteúdo
> - **@data:** "North Star: ≥ 95% em < 30s" → IA não pode adicionar latência acima de 5s ao pipeline
> - **@finance:** "Modelo recomendado por caso de uso e custo" → Gemini Flash Lite é a escolha certa para volume alto e custo baixo
> - **@finance:** "ROI de `trial.expiring` é o maior" → personalizar esta mensagem tem maior impacto financeiro
> - **@backend:** "Dispatch Engine processa em < 5s" → chamada de LLM deve ser assíncrona para não bloquear

---

## 1. Casos de Uso de IA no notification-service

Três usos distintos, com custo e latência diferentes:

| Caso de Uso | Quando | Latência Adicional | Modelo |
|------------|--------|-------------------|--------|
| Personalização de mensagem | Pré-envio | +2-4s | Gemini Flash Lite |
| Timing inteligente | Agendamento | +0s (offline) | Modelo local simples |
| Análise de opt-out | Pós-evento | Async | Gemini Flash Lite |

---

## 2. Personalização de Mensagens com Gemini Flash Lite

### 2.1 Por que Gemini Flash Lite?

| Modelo | Input Cost | Output Cost | Latência | Adequado para |
|--------|-----------|-------------|---------|---------------|
| Gemini 2.0 Flash Lite | $0.075/1M tokens | $0.30/1M tokens | ~500ms | ✅ Personalização em volume |
| Claude Haiku | $0.25/1M tokens | $1.25/1M tokens | ~600ms | Custo 3x maior |
| GPT-4o mini | $0.15/1M tokens | $0.60/1M tokens | ~800ms | Custo 2x maior |
| Gemini 1.5 Pro | $1.25/1M tokens | $5.00/1M tokens | ~1.5s | ❌ Caro para volume |

**Escolha: Gemini 2.0 Flash Lite** — melhor custo/latência para personalização de notificações transacionais em escala.

### 2.2 Custo por Notificação

Estimativa por chamada de personalização:
- Input: ~300 tokens (contexto do cliente + template base)
- Output: ~100 tokens (mensagem personalizada)
- Custo: (300 × $0.075 + 100 × $0.30) / 1M = **$0.0000525/notificação**

Com 5.000 notificações/mês personalizadas: **$0.26/mês** — custo desprezível.

### 2.3 Implementação — Personalização Seletiva

**Regra:** Não personalizar toda notificação. Priorizar as de maior ROI (@finance).

```go
// internal/ai/personalizer.go
type Personalizer struct {
    geminiClient *genai.Client
    enabled      bool
}

// Templates que SE BENEFICIAM de personalização (ordem de ROI @finance)
var personalizeableTemplates = map[string]bool{
    "trial.expiring":     true,  // maior ROI — sempre personalizar
    "payment.overdue":    true,  // recovery rate melhora com tom certo
    "class.reminder":     false, // transacional simples — não precisa
    "payment.confirmed":  false, // confirmação factual — não personalizar
    "order.ready":        false, // urgente — não adicionar latência
}

func (p *Personalizer) ShouldPersonalize(templateType string) bool {
    return p.enabled && personalizeableTemplates[templateType]
}

func (p *Personalizer) PersonalizeMessage(ctx context.Context, req PersonalizeRequest) (string, error) {
    if !p.ShouldPersonalize(req.TemplateType) {
        return req.BaseMessage, nil
    }

    prompt := fmt.Sprintf(`
Você é um assistente de comunicação para a plataforma %s.
Personalize a seguinte mensagem para o cliente.

DADOS DO CLIENTE:
- Nome: %s
- Histórico: %s aulas assistidas nos últimos 30 dias
- Frequência: %s (regular/irregular/novo)
- Tempo como cliente: %d meses

MENSAGEM BASE:
%s

REGRAS:
- Máximo 160 caracteres para WhatsApp (mensagem de negócio)
- Tom: amigável mas profissional
- Incluir o nome do cliente naturalmente
- Não inventar informações
- Não usar emoji excessivo (máx 1)
- Preservar TODOS os dados factuais (valores, datas, links)

Retorne APENAS a mensagem personalizada, sem explicações.`,
        req.TenantName,
        req.CustomerName,
        req.ClassesLast30Days,
        req.FrequencyLabel,
        req.MonthsAsCustomer,
        req.BaseMessage,
    )

    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resp, err := p.geminiClient.GenerateContent(ctx, prompt)
    if err != nil {
        // Fallback para mensagem base — nunca falhar por causa de IA
        return req.BaseMessage, nil
    }

    return resp.Text(), nil
}
```

### 2.4 Exemplos de Personalização

**trial.expiring — Sem personalização:**
> "Seu período de teste expira em 2 dias. Assine agora: link"

**trial.expiring — Com personalização (cliente regular, 8 meses):**
> "João, seu trial acaba em 2 dias! Você já participou de 12 aulas esse mês — seria uma pena parar. Assine: link"

**trial.expiring — Com personalização (cliente novo, 1 mês):**
> "Oi João! Seus 14 dias de trial acabam em 2 dias. Quer continuar treinando? link"

---

## 3. Timing Inteligente — Melhor Horário de Envio

### 3.1 Problema
Enviar `trial.expiring` às 3h da manhã → menor abertura vs. às 9h.  
Mas 9h pode não ser ótimo para todos os perfis.

### 3.2 Abordagem — Regras Simples (MVP) + ML (M6+)

**MVP (M1-M3): Regras baseadas em comportamento histórico**

```go
// internal/timing/scheduler.go
type TimingService struct {
    repo NotificationRepo
}

// Calcula melhor horário baseado em histórico de aberturas
func (t *TimingService) GetOptimalSendTime(ctx context.Context, customerID string) time.Time {
    // Buscar horários de abertura dos últimos 90 dias
    openedHours, err := t.repo.GetOpenedHours(ctx, customerID, 90)
    if err != nil || len(openedHours) < 5 {
        // Fallback: regra por segmento
        return t.getDefaultSendTime()
    }

    // Calcular hora com maior frequência de abertura
    hourCounts := make(map[int]int)
    for _, h := range openedHours {
        hourCounts[h]++
    }
    
    bestHour := 9 // default
    maxCount := 0
    for hour, count := range hourCounts {
        if count > maxCount && hour >= 7 && hour <= 21 { // janela razoável
            maxCount = count
            bestHour = hour
        }
    }

    tomorrow := time.Now().AddDate(0, 0, 1)
    return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), bestHour, 0, 0, 0, tomorrow.Location())
}

// Horários padrão por segmento (quando sem dados históricos)
func (t *TimingService) getDefaultSendTime() time.Time {
    // 9h da manhã do próximo dia útil
    now := time.Now()
    next := now.AddDate(0, 0, 1)
    if next.Weekday() == time.Saturday {
        next = next.AddDate(0, 0, 2)
    }
    return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())
}
```

**M6+: Modelo ML com dados de @data**

Quando tiver ≥ 3 meses de `opened_at` coletado (@data prioriza isso desde o dia 1):
1. Feature engineering: hora do dia, dia da semana, tipo de notificação, canal
2. Modelo simples: regressão logística de probabilidade de abertura por hora
3. Treinar por tenant (padrões diferentes entre academia e restaurante)
4. Retraining semanal com novos dados

---

## 4. Análise Inteligente de Opt-Out

### 4.1 Problema
Cliente optou out → por quê? Frequência alta? Irrelevância? Horário ruim?

### 4.2 Implementação

```go
// internal/ai/opt_out_analyzer.go
// Roda de forma ASSÍNCRONA (não no caminho crítico)
// Trigger: evento opt-out gravado na DB

func (a *OptOutAnalyzer) AnalyzeOptOut(ctx context.Context, customerID string) {
    // Buscar histórico de 30 dias antes do opt-out
    history := a.repo.GetPreOptOutHistory(ctx, customerID, 30)
    
    prompt := fmt.Sprintf(`
Analise este histórico de notificações de um cliente que acabou de fazer opt-out.
Identifique a provável causa e recomende ação corretiva.

HISTÓRICO (últimas 30 notificações antes do opt-out):
%s

Responda em JSON:
{
  "probable_cause": "frequência_alta|irrelevância|horário_ruim|outro",
  "confidence": 0.0-1.0,
  "recommendation": "reduzir_frequência|ajustar_horário|revisar_template|nenhuma_ação",
  "explanation": "breve explicação"
}`, formatHistory(history))

    resp, _ := a.geminiClient.GenerateContent(ctx, prompt)
    // Salvar análise para review do time de growth
    a.repo.SaveOptOutAnalysis(ctx, customerID, resp.Text())
}
```

Resultado exposto no admin panel (frontend @frontend): ao clicar em um opt-out, ver análise de causa provável.

---

## 5. Guardrails e Segurança

### 5.1 Sem IA no Caminho Crítico de Transacionais

```go
// Regra absoluta: payment.confirmed e order.ready NUNCA passam por IA
// Latência zero adicionada para eventos urgentes
if isUrgentTemplate(templateType) {
    return baseMessage, nil // bypass total da IA
}
```

### 5.2 Fallback Obrigatório

```go
// Se IA falha por qualquer motivo → usar template base sem IA
// NUNCA falhar a notificação por causa da IA
result, err := personalizer.PersonalizeMessage(ctx, req)
if err != nil {
    metrics.AIFallback.Inc()
    result = req.BaseMessage
}
```

### 5.3 Sem Dados Sensíveis no Prompt

- ❌ Não incluir CPF, endereço, número de cartão
- ✅ Incluir: primeiro nome, contagem de aulas, tempo como cliente
- ❌ Não incluir conversas anteriores de WhatsApp
- Revisar prompt em code review para dados LGPD (@legal)

---

## 6. Roadmap de IA

| Fase | Feature | Modelo | Impacto Estimado |
|------|---------|--------|-----------------|
| M1 | Personalização trial.expiring + payment.overdue | Gemini Flash Lite | +5% conversão |
| M3 | Timing inteligente (regras) | Lógica local | +8% open rate |
| M6 | Análise de opt-out | Gemini Flash Lite | -15% opt-out rate |
| M6 | Timing ML | TensorFlow Lite local | +12% open rate |
| M9 | Sugestão de canal por cliente (histórico de resposta) | Modelo local | -20% custo WhatsApp |
| M12 | Predição de churn pré-notificação | Gemini Flash | Integração com @growth |
