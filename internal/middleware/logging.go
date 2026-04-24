package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type key string
const requestIDKey key = "request_id"

func Logging(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := generateShortID()

			// Логгер с атрибутами запроса
			reqLogger := logger.With(
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)

			// Обёртка для перехвата статуса ответа
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			reqLogger.Debug("request started")
			next.ServeHTTP(wrapped, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))

			duration := time.Since(start)
			reqLogger.Info("request completed",
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

func generateShortID() string {
	// Простая заглушка: можно заменить на uuid.NewString()[:8]
	return fmt.Sprintf("%x", time.Now().UnixNano()%1000000)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}