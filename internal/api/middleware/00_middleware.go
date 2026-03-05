package middleware

import "net/http"

func MiddlewareWrapper(next http.Handler) http.Handler {
	next = CORS(next)
	next = Auth(next)
	return next
}
