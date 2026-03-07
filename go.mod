module github.com/institutoitinerante/notification-service

go 1.24

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-playground/validator/v10 v10.25.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/prometheus/client_golang v1.22.0
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0
	go.opentelemetry.io/otel/sdk v1.35.0
)
