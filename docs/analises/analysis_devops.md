# Analysis DevOps — notification-service

**Agente:** @devops  
**Data:** 2026-03-03  
**Serviço:** notification-service

---

## 1. Infraestrutura

### Deployment no Homelab K3s

```yaml
# k8s/notification-service/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
  namespace: iit
spec:
  replicas: 1  # MVP — single replica, escalar conforme necessidade
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
        image: ghcr.io/apsferreira/notification-service:latest
        ports:
        - containerPort: 3012
        env:
        - name: PORT
          value: "3012"
        - name: DB_URL
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: db-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: redis-url
        - name: RABBITMQ_URL
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: rabbitmq-url
        - name: RESEND_API_KEY
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: resend-api-key
        - name: META_WA_ACCESS_TOKEN
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: meta-wa-access-token
        - name: META_WA_PHONE_NUMBER_ID
          valueFrom:
            secretKeyRef:
              name: notification-service-secrets
              key: meta-wa-phone-number-id
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 3012
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 3012
          initialDelaySeconds: 5
          periodSeconds: 10
```

### Dependências de shared-infra-01

- PostgreSQL: `postgresql://notification_svc:...@shared-infra-01:5432/iit_notifications`
- Redis: DB7 (`redis://shared-infra-01:6379/7`)
- RabbitMQ: `amqp://notification_svc:...@shared-infra-01:5672/`

---

## 2. CI/CD

### GitHub Actions Pipeline

```yaml
# .github/workflows/ci.yml
name: CI/CD — notification-service

on:
  push:
    branches: [main, develop]
    paths:
      - 'notification-service/**'
      - '.github/workflows/notification-service*.yml'
  pull_request:
    branches: [main]
    paths:
      - 'notification-service/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.23'
    
    - name: Run Tests
      working-directory: notification-service
      run: |
        go test ./... -v -race -coverprofile=coverage.out
        go tool cover -func=coverage.out | tail -1
    
    - name: Check coverage ≥70%
      working-directory: notification-service
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$COVERAGE < 70" | bc -l) )); then
          echo "Coverage $COVERAGE% below 70% threshold"
          exit 1
        fi

  build-and-push:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v4
    
    - name: Login to GHCR
      uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Build and Push
      uses: docker/build-push-action@v5
      with:
        context: ./notification-service
        push: true
        tags: |
          ghcr.io/apsferreira/notification-service:latest
          ghcr.io/apsferreira/notification-service:${{ github.sha }}
    
    # ArgoCD GitOps: atualizar manifesto com novo tag
    - name: Update K8s manifest
      run: |
        sed -i "s|notification-service:.*|notification-service:${{ github.sha }}|" \
          k8s/notification-service/deployment.yaml
        git config user.name "github-actions"
        git config user.email "actions@github.com"
        git add k8s/notification-service/deployment.yaml
        git commit -m "chore: update notification-service to ${{ github.sha }}"
        git push
```

### Dockerfile Otimizado

```dockerfile
# Multi-stage build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/notification-service .
EXPOSE 3012
USER nobody
ENTRYPOINT ["./notification-service"]
```

---

## 3. Observabilidade de Delivery Rate

### Métricas Chave (Prometheus)

```go
var (
    notificationsSent = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "notifications_sent_total",
            Help: "Total de notificações enviadas",
        },
        []string{"channel", "template_type", "status"},
    )
    
    notificationDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "notification_processing_seconds",
            Help:    "Tempo de processamento de notificação",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
        },
        []string{"channel"},
    )
    
    deliveryRate = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "notification_delivery_rate",
            Help: "Taxa de entrega por canal (últimas 1h)",
        },
        []string{"channel"},
    )
)
```

### Alertas Recomendados

```yaml
# Prometheus alerting rules
groups:
- name: notification-service
  rules:
  - alert: LowDeliveryRate
    expr: notification_delivery_rate < 0.90
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Delivery rate abaixo de 90% para {{ $labels.channel }}"
  
  - alert: HighFailureRate
    expr: |
      rate(notifications_sent_total{status="failed"}[5m]) /
      rate(notifications_sent_total[5m]) > 0.10
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Taxa de falha >10% em notificações"
  
  - alert: DLQGrowing
    expr: rabbitmq_queue_messages{queue="notification-service.dlq"} > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "DLQ crescendo — {{ $value }} mensagens não processadas"
```

### Endpoints de Observabilidade

```
GET /health    → 200 se serviço está saudável
GET /ready     → 200 se conexões DB/Redis/RabbitMQ OK
GET /metrics   → Prometheus metrics
```

---

## 4. Secrets Management

Usar K3s Secrets + Sealed Secrets (ou External Secrets Operator para MVP):

```bash
# Criar secrets
kubectl create secret generic notification-service-secrets \
  --from-literal=db-url="postgresql://..." \
  --from-literal=redis-url="redis://..." \
  --from-literal=rabbitmq-url="amqp://..." \
  --from-literal=resend-api-key="re_..." \
  --from-literal=meta-wa-access-token="..." \
  --from-literal=meta-wa-phone-number-id="..." \
  -n iit
```

---

## 5. Checklist de Deploy

- [ ] Migrations rodadas (`goose up` ou `migrate up`)
- [ ] Templates de WhatsApp aprovados na Meta antes do go-live
- [ ] Secrets criados no namespace K3s
- [ ] DNS para webhook da Meta apontando para o cluster
- [ ] Alertas Prometheus configurados
- [ ] Testado end-to-end em staging com eventos reais do RabbitMQ
