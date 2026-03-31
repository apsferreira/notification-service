package middleware

import (
	"net/http"
	"os"
	"strings"
)

// ServiceTokenMiddleware verifica o token de serviço via Authorization: Bearer <token>
// ou X-Service-Token: <token>. Rejeita requests sem token válido com 401.
func ServiceTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractServiceToken(r)
		if token == "" {
			http.Error(w, `{"error":"missing service token"}`, http.StatusUnauthorized)
			return
		}

		expected := os.Getenv("SERVICE_TOKEN")
		if expected == "" {
			http.Error(w, `{"error":"service token not configured"}`, http.StatusInternalServerError)
			return
		}

		if token != expected {
			http.Error(w, `{"error":"invalid service token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractServiceToken extrai o token de Authorization: Bearer <token> ou X-Service-Token: <token>.
func extractServiceToken(r *http.Request) string {
	if xToken := r.Header.Get("X-Service-Token"); xToken != "" {
		return xToken
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
