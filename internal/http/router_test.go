package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	todohttp "github.com/jaekwang-park/todo-api/internal/http"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// mockTodoRepo for router tests
type mockTodoRepo struct{}

func (m *mockTodoRepo) Create(ctx context.Context, todo model.Todo) (model.Todo, error) {
	return model.Todo{}, nil
}
func (m *mockTodoRepo) GetByID(ctx context.Context, userID, todoID string) (model.Todo, error) {
	return model.Todo{}, fmt.Errorf("not found")
}
func (m *mockTodoRepo) Update(ctx context.Context, todo model.Todo) (model.Todo, error) {
	return model.Todo{}, nil
}
func (m *mockTodoRepo) Delete(ctx context.Context, userID, todoID string) error {
	return nil
}
func (m *mockTodoRepo) List(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
	return model.TodoListResult{Todos: []model.Todo{}}, nil
}

func newTestTodoSvc() *service.TodoService {
	return service.NewTodoService(&mockTodoRepo{})
}

func TestRouter_HealthEndpoint(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}
}

func TestRouter_TodoEndpointRegistered(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 200 with empty list, not 404
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc())

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
