# chain_06_frontend.md — @frontend
# Especificação Frontend: notification-service

**Agente:** @frontend  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_05_backend.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam as decisões de UI/UX:
>
> - **@data:** "Dashboards: Delivery Overview, Event Pipeline, Latência, Opt-out Trends, Error Analysis" → telas do admin panel
> - **@data:** "Taxa de abertura por canal (WhatsApp 80% vs Email 35%)" → exibir como KPI principal no dashboard
> - **@data:** "Opt-out rate — Alerta > 0,5% email / > 2% WhatsApp" → indicadores visuais no painel
> - **@finance:** "Custo por notificação entregue e ROI por template_type" → seção de custo no admin panel
> - **@finance:** "Budget de $200/mês é conservador" → exibir consumo atual vs. budget no admin
> - **@backend:** "GET /api/v1/notifications?customer_id=uuid&channel=whatsapp" → API para histórico

---

## 1. Escopo de Frontend

Dois contextos distintos:
1. **Admin Panel** — visão operacional para tenant admin (histórico, templates, métricas)
2. **Preferências de Canal** — self-service para o usuário final (opt-in/opt-out por canal)

Stack: **React + Vite + TypeScript** (alinhado com auth-service e demais painéis do ecossistema)

---

## 2. Admin Panel

### 2.1 Dashboard Principal

**Rota:** `/admin/notifications`

```
┌──────────────────────────────────────────────────────────────────┐
│  Notification Overview                    [Últimas 24h ▼]        │
├──────────────┬───────────────┬──────────────┬───────────────────┤
│  Entregues   │  Taxa Entrega │  Opt-outs    │  DLQ              │
│  4.832       │  97.3% ✅     │  12 (0.2%)✅ │  3 ⚠️            │
├──────────────┴───────────────┴──────────────┴───────────────────┤
│  Por Canal                                                        │
│  📧 Email    ████████████████████ 98.1%    2.340 enviados        │
│  💬 WhatsApp ████████████████░░░ 94.2%    1.890 enviados        │
│  📱 Push     █████████████░░░░░░ 82.0%      602 enviados        │
├──────────────────────────────────────────────────────────────────┤
│  Latência (evento → entrega)                                      │
│  p50: 4s   p95: 18s   p99: 28s   ← tudo dentro do SLA 30s ✅   │
└──────────────────────────────────────────────────────────────────┘
```

**Componentes:**
- `<MetricCard>` — KPI individual com indicador de tendência
- `<ChannelBar>` — barra de progresso com delivery rate por canal
- `<LatencyGauge>` — gauge de p50/p95/p99
- `<DLQBadge>` — badge vermelho se DLQ > 0

### 2.2 Histórico de Notificações por Cliente

**Rota:** `/admin/notifications/customers/:customer_id`

```
┌────────────────────────────────────────────────────────────────┐
│ ← João Silva                     [Filtrar canal ▼] [Buscar]   │
├───────┬──────────────┬──────────┬────────┬────────────────────┤
│ Canal │ Tipo         │ Status   │ Enviado│ Aberto             │
├───────┼──────────────┼──────────┼────────┼────────────────────┤
│ 💬    │ class.remind │ ✅ sent  │ 14:30  │ 14:31 (1min)       │
│ 📧    │ payment.conf │ ✅ sent  │ 10:12  │ 10:45 (33min)      │
│ 📱    │ checkin.conf │ ❌ failed│ 09:00  │ —                  │
│ 💬    │ trial.expir  │ ✅ sent  │ Ontem  │ —                  │
└───────┴──────────────┴──────────┴────────┴────────────────────┘
                                          [ Reenviar ] [ Ver payload ]
```

**Features:**
- Paginação (50 por página)
- Filtros: canal, status, data range, template_type
- Expandir linha: ver payload completo em JSON
- Botão "Reenviar": chamar `POST /api/v1/notify` com mesmo payload
- Status com cores: verde (sent/delivered), amarelo (pending), vermelho (failed)

### 2.3 Templates Manager

**Rota:** `/admin/notifications/templates`

```
┌────────────────────────────────────────────────────────────────┐
│  Templates          [+ Novo Template]                           │
├──────────────────────┬──────────┬──────────────┬──────────────┤
│ Nome                 │ Canal    │ Status Meta  │ Ações         │
├──────────────────────┼──────────┼──────────────┼──────────────┤
│ payment_confirmed    │ Email    │ N/A          │ [Editar]      │
│ payment_confirmed    │ WhatsApp │ ✅ Aprovado  │ [Editar]      │
│ class_reminder       │ WhatsApp │ ⏳ Pendente  │ [Ver]        │
│ trial_expiring       │ WhatsApp │ ❌ Rejeitado │ [Revisar]    │
└──────────────────────┴──────────┴──────────────┴──────────────┘
```

**Editor de Template:**
- Editor de texto rico para email (HTML)
- Campo de texto simples para WhatsApp (com variáveis `{{nome}}` destacadas)
- Preview ao vivo com dados de exemplo
- Indicação de status de aprovação Meta (para WhatsApp)
- Validação de variáveis obrigatórias

### 2.4 Configuração de Custo (para @finance)

**Rota:** `/admin/notifications/costs`

```
┌─────────────────────────────────────────────────────────────┐
│  Consumo do Mês — Março 2026           Budget: $200/mês     │
├──────────────────────────────────────────────────────────────┤
│  Email (Resend)                                              │
│  Enviados: 1.847 / 3.000 (free tier)    ████░░░░░  61%      │
│  Custo: $0                                                   │
├──────────────────────────────────────────────────────────────┤
│  WhatsApp (Evolution API)                                    │
│  Conversas: 423                                              │
│  Custo estimado: ~$5 (infra)                                 │
├──────────────────────────────────────────────────────────────┤
│  Push (Expo)                                                 │
│  Enviados: 892   Custo: $0                                   │
├──────────────────────────────────────────────────────────────┤
│  Total estimado: ~$5 / $200 budget    ██░░░░░░░  2.5%       │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. Preferências de Canal (Usuário Final)

### 3.1 Tela de Preferências

**Rota:** `/settings/notifications` (embedded em cada produto vertical)

```
┌───────────────────────────────────────────────────────────────┐
│  Minhas Preferências de Notificação                           │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  📧 Email                                                     │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ Confirmações de pagamento          [✅ Ativo]       │     │
│  │ Lembretes de aula                  [✅ Ativo]       │     │
│  │ Ofertas e novidades                [❌ Desativado]  │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
│  💬 WhatsApp (+55 11 99999-9999)                             │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ Confirmações de pagamento          [✅ Ativo]       │     │
│  │ Lembretes de aula                  [✅ Ativo]       │     │
│  │ Ofertas e novidades                [❌ Desativado]  │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
│  📱 Push (iPhone de João)                                     │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ Alertas de pedido pronto           [✅ Ativo]       │     │
│  │ Check-in confirmado                [✅ Ativo]       │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
│  [Salvar preferências]                                        │
└───────────────────────────────────────────────────────────────┘
```

**Regras de UX:**
- Toggle de "Confirmações de pagamento" para email/WhatsApp é sempre habilitado e não pode ser desativado (base legal: execução de contrato)
- Toggle desabilitado exibe tooltip: "Este tipo de notificação é necessário para o funcionamento do serviço"
- Salvar → `PATCH /api/v1/notifications/preferences` → feedback visual de sucesso

### 3.2 Página de Unsubscribe (Email)

**Rota:** `/unsubscribe?token=<one-time-token>&channel=email`  
**Sem autenticação** (conforme @legal e @backend)

```
┌──────────────────────────────────────────────────────────┐
│                    IIT Plataforma                        │
│                                                          │
│  Você cancelou as notificações de email ✅              │
│                                                          │
│  Não receberá mais emails promocionais.                  │
│  Notificações importantes de pagamento continuam         │
│  sendo enviadas (necessárias para o serviço).            │
│                                                          │
│  [Alterar preferências]   [Isso foi um erro, reativar]  │
└──────────────────────────────────────────────────────────┘
```

---

## 4. Componentes Reutilizáveis

```typescript
// components/notifications/StatusBadge.tsx
type Status = 'pending' | 'sent' | 'delivered' | 'failed' | 'opted_out';

// components/notifications/ChannelIcon.tsx
type Channel = 'email' | 'whatsapp' | 'push';

// components/notifications/NotificationTable.tsx
// - Filtrável, paginável, com expandir linha para payload

// components/notifications/OptOutToggle.tsx
// - Toggle com confirmação para canais que podem ser desativados
// - Tooltip para os que não podem

// hooks/useNotifications.ts
// - GET /api/v1/notifications com cache (SWR ou TanStack Query)
// - Filtros por canal, status, data
```

---

## 5. Integrações com API

```typescript
// api/notifications.ts
const api = {
  list: (params: NotificationFilters) => 
    GET('/api/v1/notifications', params),
  
  resend: (notificationId: string) =>
    POST('/api/v1/notifications/resend', { notification_id: notificationId }),
  
  getPreferences: (customerId: string) =>
    GET(`/api/v1/notifications/preferences/${customerId}`),
  
  updatePreferences: (customerId: string, prefs: Preferences) =>
    PATCH(`/api/v1/notifications/preferences/${customerId}`, prefs),
  
  getMetrics: (tenantId: string, period: string) =>
    GET(`/api/v1/notifications/metrics`, { tenant_id: tenantId, period }),
};
```

---

## 6. Acessibilidade e Internacionalização

- Labels descritivos em todos os toggles (`aria-label="Ativar notificações de email para pagamentos"`)
- Status de entrega com ícone + texto (não apenas cor)
- Timestamps em horário local do usuário (Intl.DateTimeFormat)
- Português BR como idioma único (MVP)
