package middleware

import (
	"net/http"
	"strings"
)

var publicPrefixes = []string{
	"/login",
	"/health",
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if isPublic(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		user := r.Header.Get("X-User")
		if user == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isPublic(path string) bool {
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
