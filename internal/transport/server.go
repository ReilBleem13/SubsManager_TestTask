package transport

import (
	"net/http"

	ratelimiting "github.com/ReilBleem13/internal/rateLimiting"
	"github.com/gorilla/mux"
)

type Handler interface {
	Register(mux *mux.Router, ipRateLimiter *ratelimiting.IPRateLimiter)
}

func NewServer(addr string, mux *mux.Router) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server
}

func NewRouter(ipRateLimiter *ratelimiting.IPRateLimiter, handlers ...Handler) *mux.Router {
	mux := mux.NewRouter()

	for _, h := range handlers {
		h.Register(mux, ipRateLimiter)
	}
	return mux
}
