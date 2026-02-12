package http

import (
	"net/http"

	"github.com/jaekwang-park/todo-api/internal/http/handler"
	"github.com/jaekwang-park/todo-api/internal/service"
)

func NewRouter(todoSvc *service.TodoService) http.Handler {
	mux := http.NewServeMux()

	// Health check - intentionally outside /api/v1 for ALB health check compatibility
	health := handler.NewHealthHandler()
	mux.Handle("/health", health)

	// Todo CRUD API
	todoHandler := handler.NewTodoHandler(todoSvc)
	mux.Handle("/api/v1/todos", todoHandler)
	mux.Handle("/api/v1/todos/", todoHandler)

	return mux
}
