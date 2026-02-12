package repository

import (
	"context"

	"github.com/jaekwang-park/todo-api/internal/model"
)

type TodoRepository interface {
	Create(ctx context.Context, todo model.Todo) (model.Todo, error)
	GetByID(ctx context.Context, userID, todoID string) (model.Todo, error)
	Update(ctx context.Context, todo model.Todo) (model.Todo, error)
	Delete(ctx context.Context, userID, todoID string) error
	List(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error)
}
