package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/institutoitinerante/notification-service/internal/consumer"
	"github.com/institutoitinerante/notification-service/internal/handler"
	custommw "github.com/institutoitinerante/notification-service/internal/middleware"
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
	rabbitmqURL := getEnv("RABBITMQ_URL", "")

	// OTP configuration
	telegramBotToken := getEnv("TELEGRAM_BOT_TOKEN", "")
	telegramChatID := getEnv("TELEGRAM_CHAT_ID", "")
	env := getEnv("ENV", "production")

	// WhatsApp via Meta Cloud API (opcional — no-op se credenciais ausentes)
	waPhoneNumberID := getEnv("WHATSAPP_PHONE_NUMBER_ID", "")
	waAccessToken   := getEnv("WHATSAPP_ACCESS_TOKEN", "")

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
	otpRepo := repository.NewOTPRepository(dbPool)

	// 5. Initialize services
	notificationSvc := service.NewNotificationService(notificationRepo, templateRepo, resendAPIKey, fromEmail)
	templateSvc := service.NewTemplateService(templateRepo)
	
	// Initialize OTP-related services
	emailSvc := service.NewEmailService(resendAPIKey, fromEmail, env)
	telegramNotifier := service.NewTelegramNotifier(telegramBotToken, telegramChatID)
	whatsappNotifier := service.NewMetaWhatsAppNotifier(waPhoneNumberID, waAccessToken)
	if whatsappNotifier.IsConfigured() {
		log.Println("✅ WhatsApp (Meta Cloud API) configurado")
	} else {
		log.Println("⚠️  WhatsApp não configurado — canal whatsapp desabilitado")
	}
	otpSvc := service.NewOTPService(otpRepo, emailSvc, telegramNotifier, whatsappNotifier, 10, 3, 5, 30) // 10min expiry, 3 attempts, 5 requests per 30min

	// 6. Initialize handlers
	notificationHandler := handler.NewNotificationHandler(notificationSvc)
	templateHandler := handler.NewTemplateHandler(templateSvc)
	otpHandler := handler.NewOTPHandler(otpSvc)

	// 6b. Start RabbitMQ consumers tied to the server lifecycle context.
	serverCtx, serverStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer serverStop()

	if rabbitmqURL != "" {
		handlers := consumer.NewNotificationHandlers(otpSvc, notificationSvc)
		c := consumer.New(rabbitmqURL, handlers, handlers, handlers, handlers)
		go c.Start(serverCtx)
		log.Println("✅ RabbitMQ consumer started")
	} else {
		log.Println("⚠️  RABBITMQ_URL not set — event consumers disabled")
	}

	// 7. Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(custommw.NewZerologMiddleware())
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

	r.Route("/otp", func(r chi.Router) {
		r.Post("/send", otpHandler.SendOTP)
		r.Post("/verify", otpHandler.VerifyOTP)
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

	// Wait for interrupt signal (already handled by serverCtx above)
	<-serverCtx.Done()
	serverStop()

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
