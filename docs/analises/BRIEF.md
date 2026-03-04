# Brief — notification-service

## Prioridade: 🟠 P1

## Contexto
Hub central de notificações transacionais. Consome eventos RabbitMQ de outros serviços e entrega por email, WhatsApp ou push. Complementar ao Brio (atendimento conversacional) — o notification-service faz notificações de sistema.

## Stack
Go + Fiber | Porta :3012 | RabbitMQ (consumer) + Resend (email) + Meta Cloud API (WhatsApp)

## IMPORTANTE: WhatsApp via Meta Cloud API
Não usar Evolution API — decisão do CEO. WhatsApp = Meta Cloud API apenas.

## Papel no ecossistema
- checkout.confirmed → email de recibo + WhatsApp de confirmação
- scheduling.reminder → WhatsApp 24h antes da aula
- trial.expiring → email + WhatsApp 48h antes do vencimento
- Brio escalada → Telegram para Antonio (MVP) → migrar para notification-service v1.1

## Requisitos do CEO
- REQ-NT-01: Templates por tipo de evento (não hardcoded)
- REQ-NT-02: Deduplicação (não enviar dois emails para o mesmo evento)
- REQ-NT-03: Log de envios com status (sent/delivered/failed)
- REQ-NT-04: Retry com backoff exponencial para falhas
- REQ-NT-05: WhatsApp = Meta Cloud API (não Evolution API)

## Referências
- /home/node/.openclaw/projects/docs/prd/prd_notification_service.md
- /home/node/.openclaw/workspace-main/projects-context/CONTEXT.md
