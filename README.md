# Notification Service

Microservice for sending transactional emails and SMS notifications for the Instituto Itinerante ecosystem.

## Features

- **Email sending** via [Resend](https://resend.com) API
- **Notification templates** — stored in DB with `{{variable}}` placeholder support
- **Notification log** — track all sent notifications with status
- **Retry mechanism** — failed sends retry up to 3 times with exponential backoff
- **REST API** for other services to trigger notifications
- **OpenTelemetry** tracing + **Prometheus** metrics

## Tech Stack

- Go 1.24
- Chi router
- PostgreSQL (pgx)
- Resend API (email)
- OpenTelemetry + Prometheus

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/notifications/send` | Send a notification |
| `GET` | `/notifications` | List notifications (with filters) |
| `GET` | `/notifications/{id}` | Get notification details |
| `POST` | `/templates` | Create a template |
| `GET` | `/templates` | List templates |
| `GET` | `/templates/{id}` | Get template by ID |
| `PUT` | `/templates/{id}` | Update template |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3030` | Server port |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `RESEND_API_KEY` | — | Resend API key for sending emails |
| `FROM_EMAIL` | `noreply@institutoitinerante.com.br` | Sender email address |
| `OTEL_SERVICE_NAME` | `notification-service` | OpenTelemetry service name |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP collector endpoint |

## Quick Start

### Local Development

```bash
# Start PostgreSQL and API
docker-compose up -d

# Or run directly (requires PostgreSQL running)
export DATABASE_URL="postgresql://postgres:postgres@localhost:5435/notification_service?sslmode=disable"
export RESEND_API_KEY="re_your_key_here"
go run ./cmd/server
```

### Run Migrations

Migrations are automatically applied when using docker-compose (via init scripts). For manual setup:

```bash
psql $DATABASE_URL -f migrations/001_create_templates.up.sql
psql $DATABASE_URL -f migrations/002_create_notifications.up.sql
```

## Usage Examples

### Create a Template

```bash
curl -X POST http://localhost:3030/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "welcome_email",
    "type": "email",
    "subject_template": "Welcome, {{customer_name}}!",
    "body_template": "<h1>Hello {{customer_name}}</h1><p>Welcome to Instituto Itinerante!</p>"
  }'
```

### Send a Notification with Template

```bash
curl -X POST http://localhost:3030/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "recipient": "user@example.com",
    "template_id": "TEMPLATE_UUID_HERE",
    "variables": {
      "customer_name": "João"
    }
  }'
```

### Send a Direct Notification

```bash
curl -X POST http://localhost:3030/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email",
    "recipient": "user@example.com",
    "subject": "Order Confirmation",
    "body": "<h1>Your order has been confirmed!</h1>"
  }'
```

## Project Structure

```
notification-service/
├── cmd/server/main.go          # Entry point
├── internal/
│   ├── handler/                # HTTP handlers
│   ├── model/                  # Data models
│   ├── repository/             # Database access
│   ├── service/                # Business logic
│   └── telemetry/              # OpenTelemetry + Prometheus
├── migrations/                 # SQL migrations
├── deploy/
│   ├── k8s/                    # Kubernetes manifests
│   └── argocd/                 # ArgoCD application
├── .github/workflows/ci.yml   # CI/CD pipeline
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## Deployment

- **Container Registry:** `ghcr.io/apsferreira/notification-service`
- **Domain:** `notification-api.institutoitinerante.com.br`
- **CI/CD:** GitHub Actions → GHCR → ArgoCD GitOps
