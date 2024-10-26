package middlewares

import (
	"net/http"

	"golang.org/x/time/rate"
)

// RateLimitMiddleware limits the number of requests to prevent abuse.
func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(1, 3) // 1 request per second with a burst of 3

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
