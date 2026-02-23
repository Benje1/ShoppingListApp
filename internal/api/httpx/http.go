package httpx

import (
	"encoding/json"
	"net/http"
)

type AppHandler func(r *http.Request) (any, error)

func Wrap(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := h(r)
		if err != nil {
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
