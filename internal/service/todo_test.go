package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// mockTodoRepo implements repository.TodoRepository for testing
type mockTodoRepo struct {
	createFn func(ctx context.Context, todo model.Todo) (model.Todo, error)
	getByIDFn func(ctx context.Context, userID, todoID string) (model.Todo, error)
	updateFn func(ctx context.Context, todo model.Todo) (model.Todo, error)
	deleteFn func(ctx context.Context, userID, todoID string) error
	listFn   func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error)
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

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		input   service.CreateTodoInput
		repoErr error
		wantErr string
	}{
		{
			name:    "success",
			input:   service.CreateTodoInput{Title: "Buy groceries", Description: "Milk"},
			repoErr: nil,
			wantErr: "",
		},
		{
			name:    "empty title",
			input:   service.CreateTodoInput{Title: ""},
			repoErr: nil,
			wantErr: "invalid input",
		},
		{
			name:    "repo error",
			input:   service.CreateTodoInput{Title: "Buy groceries"},
			repoErr: fmt.Errorf("db error"),
			wantErr: "failed to create todo",
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
			svc := service.NewTodoService(repo)
			got, err := svc.Create(context.Background(), "user-1", tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !containsStr(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Title != tt.input.Title {
				t.Errorf("expected title=%q, got %q", tt.input.Title, got.Title)
			}
			if got.Status != model.TodoStatusPending {
				t.Errorf("expected status=pending, got %s", got.Status)
			}
		})
	}
}

func TestGetByID(t *testing.T) {
	tests := []struct {
		name    string
		repoFn func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantErr error
	}{
		{
			name: "success",
			repoFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantErr: nil,
		},
		{
			name: "not found",
			repoFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return model.Todo{}, fmt.Errorf("failed to scan todo: %w", sql.ErrNoRows)
			},
			wantErr: service.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{getByIDFn: tt.repoFn}
			svc := service.NewTodoService(repo)
			got, err := svc.GetByID(context.Background(), "user-1", "todo-1")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != "todo-1" {
				t.Errorf("expected id=todo-1, got %s", got.ID)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	title := "Updated title"
	desc := "Updated desc"
	emptyTitle := ""

	tests := []struct {
		name    string
		input   service.UpdateTodoInput
		getFn   func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantErr string
	}{
		{
			name:  "success update title",
			input: service.UpdateTodoInput{Title: &title},
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantErr: "",
		},
		{
			name:  "success update description",
			input: service.UpdateTodoInput{Description: &desc},
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantErr: "",
		},
		{
			name:  "empty title",
			input: service.UpdateTodoInput{Title: &emptyTitle},
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantErr: "invalid input",
		},
		{
			name:  "not found",
			input: service.UpdateTodoInput{Title: &title},
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return model.Todo{}, fmt.Errorf("scan: %w", sql.ErrNoRows)
			},
			wantErr: "not found",
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
			svc := service.NewTodoService(repo)
			got, err := svc.Update(context.Background(), "user-1", "todo-1", tt.input)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !containsStr(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.input.Title != nil && got.Title != *tt.input.Title {
				t.Errorf("expected title=%q, got %q", *tt.input.Title, got.Title)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{"success", nil, nil},
		{"not found", sql.ErrNoRows, service.ErrNotFound},
		{"repo error", fmt.Errorf("db error"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				deleteFn: func(ctx context.Context, userID, todoID string) error {
					return tt.repoErr
				},
			}
			svc := service.NewTodoService(repo)
			err := svc.Delete(context.Background(), "user-1", "todo-1")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}

			if tt.repoErr != nil && !errors.Is(tt.repoErr, sql.ErrNoRows) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if tt.repoErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  model.TodoStatus
		getFn   func(ctx context.Context, userID, todoID string) (model.Todo, error)
		wantErr string
	}{
		{
			name:   "mark completed",
			status: model.TodoStatusCompleted,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return sampleTodo(), nil
			},
			wantErr: "",
		},
		{
			name:   "mark pending",
			status: model.TodoStatusPending,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				todo := sampleTodo()
				todo.Status = model.TodoStatusCompleted
				return todo, nil
			},
			wantErr: "",
		},
		{
			name:    "invalid status",
			status:  model.TodoStatus("invalid"),
			getFn:   nil,
			wantErr: "invalid input",
		},
		{
			name:   "not found",
			status: model.TodoStatusCompleted,
			getFn: func(ctx context.Context, userID, todoID string) (model.Todo, error) {
				return model.Todo{}, fmt.Errorf("scan: %w", sql.ErrNoRows)
			},
			wantErr: "not found",
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
			svc := service.NewTodoService(repo)
			got, err := svc.UpdateStatus(context.Background(), "user-1", "todo-1", tt.status)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !containsStr(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.status {
				t.Errorf("expected status=%s, got %s", tt.status, got.Status)
			}
		})
	}
}

func TestList(t *testing.T) {
	statusPending := model.TodoStatusPending

	tests := []struct {
		name    string
		params  model.TodoListParams
		result  model.TodoListResult
		repoErr error
		wantErr bool
	}{
		{
			name:   "success no filter",
			params: model.TodoListParams{UserID: "user-1", Limit: 20},
			result: model.TodoListResult{
				Todos: []model.Todo{sampleTodo()},
			},
		},
		{
			name:   "success with status filter",
			params: model.TodoListParams{UserID: "user-1", Status: &statusPending, Limit: 20},
			result: model.TodoListResult{
				Todos: []model.Todo{sampleTodo()},
			},
		},
		{
			name:   "success with cursor",
			params: model.TodoListParams{UserID: "user-1", Cursor: "cursor-id", Limit: 20},
			result: model.TodoListResult{
				Todos:      []model.Todo{sampleTodo()},
				NextCursor: "next-id",
			},
		},
		{
			name:    "repo error",
			params:  model.TodoListParams{UserID: "user-1", Limit: 20},
			repoErr: fmt.Errorf("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTodoRepo{
				listFn: func(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
					if tt.repoErr != nil {
						return model.TodoListResult{}, tt.repoErr
					}
					return tt.result, nil
				},
			}
			svc := service.NewTodoService(repo)
			got, err := svc.List(context.Background(), tt.params)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Todos) != len(tt.result.Todos) {
				t.Errorf("expected %d todos, got %d", len(tt.result.Todos), len(got.Todos))
			}
			if got.NextCursor != tt.result.NextCursor {
				t.Errorf("expected cursor=%q, got %q", tt.result.NextCursor, got.NextCursor)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
