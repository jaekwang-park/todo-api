package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaekwang-park/todo-api/internal/http/handler"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// mockTodoRepo for handler tests
type mockTodoRepo struct {
	createFn  func(ctx context.Context, todo model.Todo) (model.Todo, error)
	getByIDFn func(ctx context.Context, userID, todoID string) (model.Todo, error)
	updateFn  func(ctx context.Context, todo model.Todo) (model.Todo, error)
	deleteFn  func(ctx context.Context, userID, todoID string) error
	listFn    func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error)
}

func (m *mockTodoRepo) Create(ctx context.Context, todo model.Todo) (model.Todo, error) {
	return m.createFn(ctx, todo)
}
func (m *mockTodoRepo) GetByID(ctx context.Context, userID, todoID string) (model.Todo, error) {
	return m.getByIDFn(ctx, userID, todoID)
}
func (m *mockTodoRepo) Update(ctx context.Context, todo model.Todo) (model.Todo, error) {
	return m.updateFn(ctx, todo)
}
func (m *mockTodoRepo) Delete(ctx context.Context, userID, todoID string) error {
	return m.deleteFn(ctx, userID, todoID)
}
func (m *mockTodoRepo) List(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
	return m.listFn(ctx, params)
}

var now = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func sampleTodo() model.Todo {
	return model.Todo{
		ID:          "todo-1",
		UserID:      "user-1",
		Title:       "Buy groceries",
		Description: "Milk, eggs, bread",
		Status:      model.TodoStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func newTodoHandler(repo *mockTodoRepo) *handler.TodoHandler {
	svc := service.NewTodoService(repo)
	return handler.NewTodoHandler(svc)
}

func TestTodoHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		repoErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"title":"Buy groceries","description":"Milk"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty title",
			body:       `{"title":"","description":"Milk"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repo error",
			body:       `{"title":"Buy groceries"}`,
			repoErr:    fmt.Errorf("db error"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				createFn: func(ctx context.Context, todo model.Todo) (model.Todo, error) {
					if tt.repoErr != nil {
						return model.Todo{}, tt.repoErr
					}
					result := sampleTodo()
					result.Title = todo.Title
					result.Description = todo.Description
					return result, nil
				},
			}

			h := newTodoHandler(repo)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewBufferString(tt.body))
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var result model.Todo
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode: %v", err)
				}
				if result.Title != "Buy groceries" {
					t.Errorf("expected title=Buy groceries, got %s", result.Title)
				}
			}
		})
	}
}

func TestTodoHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		todoID     string
		repoFn     func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantStatus int
	}{
		{
			name:   "success",
			todoID: "todo-1",
			repoFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "not found",
			todoID: "nonexistent",
			repoFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return model.Todo{}, fmt.Errorf("scan: %w", sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{getByIDFn: tt.repoFn}
			h := newTodoHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+tt.todoID, nil)
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestTodoHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		getFn      func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"title":"Updated title"}`,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			getFn:      nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			body: `{"title":"Updated"}`,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return model.Todo{}, fmt.Errorf("scan: %w", sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				getByIDFn: tt.getFn,
				updateFn: func(ctx context.Context, todo model.Todo) (model.Todo, error) {
					return todo, nil
				},
			}
			h := newTodoHandler(repo)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/todo-1", bytes.NewBufferString(tt.body))
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestTodoHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		repoErr    error
		wantStatus int
	}{
		{"success", nil, http.StatusNoContent},
		{"not found", sql.ErrNoRows, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				deleteFn: func(ctx context.Context, userID, todoID string) error {
					return tt.repoErr
				},
			}
			h := newTodoHandler(repo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/todo-1", nil)
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestTodoHandler_UpdateStatus(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		getFn      func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantStatus int
	}{
		{
			name:   "mark completed",
			method: http.MethodPatch,
			body:   `{"status":"completed"}`,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid method",
			method:     http.MethodPost,
			body:       `{"status":"completed"}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid status",
			method: http.MethodPatch,
			body:   `{"status":"invalid"}`,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			method:     http.MethodPatch,
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				getByIDFn: tt.getFn,
				updateFn: func(ctx context.Context, todo model.Todo) (model.Todo, error) {
					return todo, nil
				},
			}
			h := newTodoHandler(repo)

			req := httptest.NewRequest(tt.method, "/api/v1/todos/todo-1/status", bytes.NewBufferString(tt.body))
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestTodoHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		listFn     func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error)
		wantStatus int
	}{
		{
			name:  "success no filter",
			query: "",
			listFn: func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
				return model.TodoListResult{Todos: []model.Todo{sampleTodo()}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "with status filter",
			query: "?status=pending",
			listFn: func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
				if params.Status == nil || *params.Status != model.TodoStatusPending {
					return model.TodoListResult{}, fmt.Errorf("expected status filter pending")
				}
				return model.TodoListResult{Todos: []model.Todo{sampleTodo()}}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid status filter",
			query:      "?status=invalid",
			listFn:     nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "with cursor and limit",
			query: "?cursor=abc&limit=10",
			listFn: func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
				if params.Cursor != "abc" || params.Limit != 10 {
					return model.TodoListResult{}, fmt.Errorf("expected cursor=abc, limit=10")
				}
				return model.TodoListResult{Todos: []model.Todo{}, NextCursor: "def"}, nil
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				listFn: tt.listFn,
			}
			h := newTodoHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/todos"+tt.query, nil)
			req.Header.Set("X-User-ID", "user-1")
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestTodoHandler_MethodNotAllowed(t *testing.T) {
	repo := &mockTodoRepo{}
	h := newTodoHandler(repo)

	// PATCH on collection
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/todos", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
