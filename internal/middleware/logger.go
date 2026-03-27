package middleware

import (
	"net/http"
	"os"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

// NewZerologMiddleware retorna um middleware Chi que registra cada request em JSON estruturado via zerolog.
func NewZerologMiddleware() func(next http.Handler) http.Handler {
	level := os.Getenv("LOG_LEVEL")
	lvl, err := zerolog.ParseLevel(level)
	if err != nil || level == "" {
		lvl = zerolog.InfoLevel
	}

	log := zerolog.New(os.Stdout).
		Level(lvl).
		With().
		Timestamp().
		Str("service", "notification-service").
		Logger()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Usa o RequestID do chi middleware
			requestID := chimw.GetReqID(r.Context())

			// Wrap response writer para capturar status code
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

			// Processa request
			next.ServeHTTP(ww, r)

			latency := time.Since(start)
			status := ww.Status()

			event := log.Info()
			if status >= 500 {
				event = log.Error()
			} else if status >= 400 {
				event = log.Warn()
			}

			event.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", status).
				Dur("latency", latency).
				Str("request_id", requestID).
				Str("ip", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("request")
		})
	}
}
