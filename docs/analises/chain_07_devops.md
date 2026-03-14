# chain_07_devops.md — @devops
# Infraestrutura e Deploy: notification-service

**Agente:** @devops  
**Data:** 2026-03-03  
**Leu:** chain_01_data.md, chain_02_finance.md, chain_05_backend.md

---

## Alinhamento com @data e @finance

> Citações diretas que guiam as decisões de infraestrutura:
>
> - **@data:** "North Star: ≥ 95% eventos críticos entregues em < 30s" → SLA que define sizing de infra e alertas
> - **@data:** "Alertas operacionais: DLQ > 50 msgs → alerta crítico; Latência p95 > 30s → alerta" → thresholds para alerting
> - **@data:** "Alertas: dead-letter queue crescendo, taxa de entrega caindo" → configurar Prometheus + Alertmanager
> - **@finance:** "Custo total esperado M3: ~$5/mês de infra" → não over-provisionar; K3s já existente
> - **@finance:** "Evolution API: custo de infra ~$5/mês para 2.000 conversas" → 1 pod leve é suficiente para MVP
> - **@backend:** "Redis DB7, RabbitMQ exchanges específicos, filas com DLQ" → configuração necessária no setup

---

## 1. Ambiente de Deploy

### Dev: Docker Compose (atual)
### Produção: Proxmox + K3s (planejado)

O notification-service roda no namespace `shared-services` (alinhado com arquitetura do ecossistema).

---

## 2. Dockerfile

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/server/main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/notification-service .
COPY --from=builder /app/internal/templates ./internal/templates
EXPOSE 3012
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3012/health || exit 1
USER nobody
ENTRYPOINT ["./notification-service"]
```

---

## 3. Docker Compose (Dev)

```yaml
# notification-service/docker-compose.yml
version: '3.8'

services:
  notification-service:
    build: .
    container_name: notification-service
    ports:
      - "3012:3012"
    env_file: .env
    networks:
      - shared-infra
    depends_on:
      - shared-postgres
      - shared-redis
      - shared-rabbitmq
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3012/health"]
      interval: 30s
      timeout: 5s
      retries: 3

networks:
  shared-infra:
    external: true
```

---

## 4. Kubernetes (K3s) Manifests

### 4.1 Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
  namespace: shared-services
spec:
  replicas: 2  # HA mínima
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
      - name: notification-service
        image: registry.iit.local/notification-service:latest
        ports:
        - containerPort: 3012
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
        envFrom:
        - secretRef:
            name: notification-service-secrets
        - configMapRef:
            name: notification-service-config
        livenessProbe:
          httpGet:
            path: /health
            port: 3012
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 3012
          initialDelaySeconds: 5
          periodSeconds: 10
```

### 4.2 Service + Ingress

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: notification-service
  namespace: shared-services
spec:
  selector:
    app: notification-service
  ports:
  - port: 3012
    targetPort: 3012
---
# k8s/ingress.yaml (Traefik)
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: notification-service
  namespace: shared-services
  annotations:
    traefik.ingress.kubernetes.io/router.middlewares: shared-services-auth@kubernetescrd
spec:
  rules:
  - host: api.iit.local
    http:
      paths:
      - path: /api/v1/notifications
        pathType: Prefix
        backend:
          service:
            name: notification-service
            port:
              number: 3012
      - path: /api/v1/notify
        pathType: Prefix
        backend:
          service:
            name: notification-service
            port:
              number: 3012
```

---

## 5. Secrets

```yaml
# k8s/secrets.yaml (NUNCA commitar — usar Sealed Secrets ou Vault)
apiVersion: v1
kind: Secret
metadata:
  name: notification-service-secrets
  namespace: shared-services
type: Opaque
stringData:
  DATABASE_URL: "postgres://notif_user:xxx@shared-postgres:5432/notification_db"
  REDIS_URL: "redis://shared-redis:6379/7"
  RABBITMQ_URL: "amqp://user:pass@shared-rabbitmq:5672"
  RESEND_API_KEY: "re_xxx"
  EVOLUTION_API_KEY: "xxx"
  META_ACCESS_TOKEN: ""  # vazio até migrar
  SERVICE_TOKEN: "xxx"
```

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: notification-service-config
  namespace: shared-services
data:
  PORT: "3012"
  EVOLUTION_API_URL: "http://shared-evolution-api:8081"
  EVOLUTION_INSTANCE: "iit-main"
  REDIS_DEDUP_TTL_HOURS: "24"
  MAX_RETRY_ATTEMPTS: "3"
```

---

## 6. Configuração RabbitMQ

Script de setup (rodar uma vez na inicialização):

```bash
#!/bin/bash
# scripts/rabbitmq-setup.sh

RABBIT_URL="http://user:pass@shared-rabbitmq:15672/api"

# Declarar exchanges (se não existirem)
exchanges=("checkout.events" "scheduling.events" "order.events" "table.events")
for exchange in "${exchanges[@]}"; do
  curl -s -X PUT "$RABBIT_URL/exchanges/%2F/$exchange" \
    -H "Content-Type: application/json" \
    -d '{"type":"fanout","durable":true}' 
done

# Declarar Dead-Letter Exchange
curl -s -X PUT "$RABBIT_URL/exchanges/%2F/notification.dlx" \
  -H "Content-Type: application/json" \
  -d '{"type":"direct","durable":true}'

# Declarar DLQ
curl -s -X PUT "$RABBIT_URL/queues/%2F/notification.dlq" \
  -H "Content-Type: application/json" \
  -d '{"durable":true}'

# Declarar filas com DLQ configurado
queues=(
  "notification.payment.confirmed:checkout.events"
  "notification.class.reminder:scheduling.events"
  "notification.order.ready:order.events"
  "notification.trial.expiring:checkout.events"
  "notification.payment.overdue:checkout.events"
  "notification.waiter.requested:table.events"
)

for entry in "${queues[@]}"; do
  queue="${entry%%:*}"
  exchange="${entry##*:}"
  
  # Criar fila com DLX
  curl -s -X PUT "$RABBIT_URL/queues/%2F/$queue" \
    -H "Content-Type: application/json" \
    -d "{\"durable\":true,\"arguments\":{\"x-dead-letter-exchange\":\"notification.dlx\",\"x-message-ttl\":86400000}}"
  
  # Bind fila → exchange
  curl -s -X POST "$RABBIT_URL/bindings/%2F/e/$exchange/q/$queue" \
    -H "Content-Type: application/json" \
    -d '{"routing_key":""}'
done

echo "RabbitMQ setup completo!"
```

---

## 7. Monitoramento e Alertas

### 7.1 Métricas Prometheus (expostas no /metrics)

```go
// Métricas a instrumentar no código (@data: North Star e alertas)
var (
    notificationsSent = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "notifications_sent_total",
        Help: "Total de notificações enviadas",
    }, []string{"channel", "template_type", "status"})

    notificationLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "notification_delivery_latency_seconds",
        Help:    "Latência evento → entrega",
        Buckets: []float64{1, 5, 10, 30, 60, 120},
    }, []string{"channel"})

    dlqSize = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "notification_dlq_size",
        Help: "Número de mensagens na DLQ",
    })

    optOutTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "notification_opt_out_total",
    }, []string{"channel"})
)
```

### 7.2 Alertas (Prometheus Alertmanager)

```yaml
# monitoring/alerts.yaml
groups:
- name: notification-service
  rules:
  
  # DLQ crescendo — alerta crítico (threshold de @data)
  - alert: NotificationDLQHigh
    expr: notification_dlq_size > 50
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "DLQ com {{ $value }} mensagens"
      description: "Dead-letter queue acima de 50. Verificar logs do notification-service."

  # Taxa de entrega caindo
  - alert: NotificationDeliveryRateLow
    expr: |
      rate(notifications_sent_total{status="sent"}[5m]) /
      rate(notifications_sent_total[5m]) < 0.90
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Taxa de entrega abaixo de 90% por 5 minutos"

  # Latência p95 > 30s (North Star de @data)
  - alert: NotificationLatencyHigh
    expr: |
      histogram_quantile(0.95, rate(notification_delivery_latency_seconds_bucket[5m])) > 30
    for: 3m
    labels:
      severity: warning
    annotations:
      summary: "p95 de latência acima de 30s"

  # Opt-out rate alto (threshold de @data)
  - alert: WhatsAppOptOutRateHigh
    expr: |
      rate(notification_opt_out_total{channel="whatsapp"}[1h]) /
      rate(notifications_sent_total{channel="whatsapp"}[1h]) > 0.02
    for: 0m
    labels:
      severity: warning
    annotations:
      summary: "Taxa de opt-out WhatsApp > 2%"

  # Service down
  - alert: NotificationServiceDown
    expr: up{job="notification-service"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "notification-service está down"
```

### 7.3 Destino de Alertas

Enviar para canal Telegram do admin via Alertmanager webhook (usando o próprio notification-service é irônico — usar Telegram direto via bot token para alertas de infra).

---

## 8. CI/CD Pipeline

```yaml
# .github/workflows/deploy.yaml
name: Deploy notification-service

on:
  push:
    branches: [main]
    paths: ['notification-service/**']

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - name: Run tests with coverage
        run: |
          cd notification-service
          go test ./... -coverprofile=coverage.out -covermode=atomic
      - name: Check 70% coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            echo "Coverage $COVERAGE% below 70%"
            exit 1
          fi

  build-and-deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Build Docker image
        run: docker build -t registry.iit.local/notification-service:${{ github.sha }} .
      - name: Push to registry
        run: docker push registry.iit.local/notification-service:${{ github.sha }}
      - name: Deploy to K3s
        run: |
          kubectl set image deployment/notification-service \
            notification-service=registry.iit.local/notification-service:${{ github.sha }} \
            -n shared-services
          kubectl rollout status deployment/notification-service -n shared-services
```

---

## 9. Runbook de Operações

### Verificar saúde do serviço
```bash
kubectl get pods -n shared-services -l app=notification-service
kubectl logs -n shared-services -l app=notification-service --tail=100
```

### Ver DLQ
```bash
# Via RabbitMQ Management UI: http://shared-rabbitmq:15672
# Fila: notification.dlq
curl -u user:pass http://shared-rabbitmq:15672/api/queues/%2F/notification.dlq
```

### Reprocessar DLQ manualmente
```bash
# Mover mensagens da DLQ de volta para a fila original
# Via RabbitMQ shovel plugin ou script dedicado
kubectl exec -n shared-services deploy/notification-service -- \
  ./notification-service --reprocess-dlq
```
