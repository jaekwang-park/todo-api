package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jaekwang-park/todo-api/internal/middleware"
	"github.com/jaekwang-park/todo-api/internal/service"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(port string, logger *slog.Logger, todoSvc *service.TodoService) *Server {
	router := NewRouter(todoSvc)

	// Apply middleware chain: recovery -> logging -> router
	chain := middleware.Recovery(logger)(middleware.Logging(logger)(router))

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      chain,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("starting server", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
