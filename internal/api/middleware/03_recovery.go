package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"weekly-shopping-app/internal/logger"
)

// Recovery catches any panic in a downstream handler, logs the stack trace,
// and returns a 500 so the server process keeps running.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					slog.Any("error", rec),
					slog.String("stack", string(debug.Stack())),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
