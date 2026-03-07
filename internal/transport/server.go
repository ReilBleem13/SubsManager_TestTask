package transport

import "net/http"

type Handler interface {
	Register(mux *http.ServeMux)
}

func NewServer(addr string, mux *http.ServeMux) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return server
}

func NewRouter(handlers ...Handler) *http.ServeMux {
	mux := http.NewServeMux()
	for _, h := range handlers {
		h.Register(mux)
	}
	return mux
}
