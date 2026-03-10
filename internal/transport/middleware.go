package transport

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ReilBleem13/internal/domain"
	ratelimiting "github.com/ReilBleem13/internal/rateLimiting"
	"github.com/ReilBleem13/internal/utils"
	"github.com/rs/xid"
)

const (
	requestIDHeader = "X-Request-ID"
)

type Middleware func(http.Handler) http.Handler

func conveyor(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}

func requestIDMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestID := r.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = xid.New().String()
		}

		ctx = utils.SetRequestID(ctx, requestID)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			logger.InfoContext(r.Context(), "Request start",
				"method", r.Method,
				"path", r.URL.Path,
			)

			h.ServeHTTP(w, r)

			logger.InfoContext(r.Context(), "Request finished", "time_per_request", time.Since(start).Milliseconds())
		})
	}
}

func rateLimitMiddleware(ipRateLimiter *ratelimiting.IPRateLimiter) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				handleError(w, domain.ErrInternalServer().WithMessage("invalid IP"))
				return
			}

			limiter := ipRateLimiter.GetLimiter(ip)
			if limiter.Allow() {
				h.ServeHTTP(w, r)
			} else {
				handleError(w, domain.ErrTooManyRequests().WithMessage("rate limit exceeded"))
			}
		})
	}
}
