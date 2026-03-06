package middleware

import (
	"net/http"
	"strings"

	"weekly-shopping-app/authentication"
)

var publicPrefixes = []string{
	"/login",
	"/logout",
	"/health",
	"/users/create",
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Allow public paths without auth
		if isPublic(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check session from cookie
		user, ok := authentication.GetUser(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject user into header for downstream handlers (optional)
		r.Header.Set("X-User", user)

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
