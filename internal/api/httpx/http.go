package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"weekly-shopping-app/internal/logger"
)

type AppHandler func(w http.ResponseWriter, r *http.Request) (any, error)

// Wrap converts an AppHandler into a standard http.HandlerFunc.
// Any error returned by the handler is:
//   - logged as a structured ERROR with method, path, and the error message
//   - returned to the client as a 400 JSON response
func Wrap(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := h(w, r)
		if err != nil {
			logger.Error("request error",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			JSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
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
