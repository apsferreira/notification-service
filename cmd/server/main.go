package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/institutoitinerante/notification-service/internal/handler"
	"github.com/institutoitinerante/notification-service/internal/repository"
	"github.com/institutoitinerante/notification-service/internal/service"
	"github.com/institutoitinerante/notification-service/internal/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Initialize telemetry
	shutdownTelemetry := telemetry.Init("notification-service")
	defer shutdownTelemetry()

	// 2. Load environment variables
	port := getEnv("PORT", "3030")
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	resendAPIKey := getEnv("RESEND_API_KEY", "")
	if resendAPIKey == "" {
		log.Println("⚠️  RESEND_API_KEY not set — email sending will fail")
	}
	fromEmail := getEnv("FROM_EMAIL", "noreply@institutoitinerante.com.br")

	// 3. Connect to PostgreSQL
	dbPool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Test database connection
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ PostgreSQL connected")

	// 4. Initialize repositories
	notificationRepo := repository.NewNotificationRepository(dbPool)
	templateRepo := repository.NewTemplateRepository(dbPool)

	// 5. Initialize services
	notificationSvc := service.NewNotificationService(notificationRepo, templateRepo, resendAPIKey, fromEmail)
	templateSvc := service.NewTemplateService(templateRepo)

	// 6. Initialize handlers
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	templateHandler := handler.NewTemplateHandler(templateSvc)

	// 7. Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Register telemetry endpoints
	telemetry.RegisterMetrics(r)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "notification-service"}`))
	})

	// API routes
	r.Route("/notifications", func(r chi.Router) {
		r.Post("/send", notificationHandler.SendNotification)
		r.Get("/", notificationHandler.ListNotifications)
		r.Get("/{id}", notificationHandler.GetNotification)
	})

	r.Route("/templates", func(r chi.Router) {
		r.Post("/", templateHandler.CreateTemplate)
		r.Get("/", templateHandler.ListTemplates)
		r.Get("/{id}", templateHandler.GetTemplate)
		r.Put("/{id}", templateHandler.UpdateTemplate)
	})

	// 8. Start server
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("🚀 Notification Service starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server shut down gracefully")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
