package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"weekly-shopping-app/internal/logger"
)

// ClientError wraps an error that should be reported to the caller as a 4xx
// response. Wrap a plain error with NewClientError to opt out of the default
// 500 behaviour.
type ClientError struct{ err error }

func NewClientError(err error) ClientError    { return ClientError{err} }
func (e ClientError) Error() string           { return e.err.Error() }
func (e ClientError) Unwrap() error           { return e.err }

type AppHandler func(w http.ResponseWriter, r *http.Request) (any, error)

// Wrap converts an AppHandler into a standard http.HandlerFunc.
// Errors that are (or wrap) ClientError are returned as 400; all others are
// logged and returned as 500 so internal details are never leaked to callers.
func Wrap(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := h(w, r)
		if err != nil {
			var ce ClientError
			if errors.As(err, &ce) {
				JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			} else {
				logger.Error("internal error",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("error", err.Error()),
				)
				JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{
		"error": msg,
	})
}
