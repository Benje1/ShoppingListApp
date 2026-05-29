package middleware

import "net/http"

func MiddlewareWrapper(next http.Handler) http.Handler {
	next = CORS(next)
	next = LoginRateLimiter()(next)
	next = RequestLogger(next)
	next = Recovery(next)
	return next
}
