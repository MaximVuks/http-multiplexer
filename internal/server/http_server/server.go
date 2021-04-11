package http_server

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

type httpServer struct {
	port           int
	logger         *log.Logger
	errorChan      chan<- error
	router         *http.ServeMux
	handlers       map[string]http.HandlerFunc
	server         *http.Server
	maxConnections int
	connections    chan struct{}
}

func New(
	ctx context.Context,
	port int,
	logger *log.Logger,
	errorChan chan<- error,
	maxConnections int,
	handlers map[string]http.HandlerFunc,
) *httpServer {
	router := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: router,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	return &httpServer{
		port:           port,
		logger:         logger,
		errorChan:      errorChan,
		router:         router,
		handlers:       handlers,
		server:         server,
		maxConnections: maxConnections,
		connections:    make(chan struct{}, maxConnections),
	}
}

func (s *httpServer) Run() {
	for method, handle := range s.handlers {
		s.router.Handle(method, s.maxConnectionsMiddleware(handle))
	}

	s.router.HandleFunc("/healthcheck", func(writer http.ResponseWriter, request *http.Request) {})

	go func() {
		s.logger.Println("http server started on [::]:" + strconv.Itoa(s.port))

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.errorChan <- err
		}
	}()

}

func (s *httpServer) Stop() error {
	s.logger.Println("http server is shutting down...")

	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*25)
	defer ctxCancel()

	s.server.SetKeepAlivesEnabled(false)
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func (s *httpServer) maxConnectionsMiddleware(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.connections) >= s.maxConnections {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		s.connections <- struct{}{}
		defer func() {
			<-s.connections
		}()

		nextHandler.ServeHTTP(w, r)
	})
}
