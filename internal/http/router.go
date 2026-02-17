package http

import (
	"net/http"

	"github.com/jaekwang-park/todo-api/internal/http/handler"
	"github.com/jaekwang-park/todo-api/internal/service"
)

func NewRouter(todoSvc *service.TodoService, authSvc *service.AuthService) http.Handler {
	mux := http.NewServeMux()

	// Health check - intentionally outside /api/v1 for ALB health check compatibility
	health := handler.NewHealthHandler()
	mux.Handle("/health", health)

	// Auth endpoints (no JWT middleware needed)
	authHandler := handler.NewAuthHandler(authSvc)
	mux.Handle("/api/v1/auth/", authHandler)

	// Todo CRUD API
	todoHandler := handler.NewTodoHandler(todoSvc)
	mux.Handle("/api/v1/todos", todoHandler)
	mux.Handle("/api/v1/todos/", todoHandler)

	return mux
}
