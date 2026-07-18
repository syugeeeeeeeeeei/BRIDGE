package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func (s *Server) requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" {
			id = fmt.Sprintf("req-%016x", s.requestID.Add(1))
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}
func requestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, category, msg, requestID string) {
	writeJSON(w, status, map[string]any{"schema_version": "bridge.error.v1", "error": map[string]any{"code": code, "message": msg, "category": category, "retryable": status == http.StatusTooManyRequests || status >= 500, "request_id": requestID}})
}
func recoverer(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				logger.Error("panic recovered", "panic", fmt.Sprint(v), "request_id", requestIDFrom(r.Context()))
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal", "内部エラーが発生しました", requestIDFrom(r.Context()))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
