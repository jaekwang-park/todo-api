package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jaekwang-park/todo-api/internal/model"
)

type PostgresTodoRepository struct {
	db *sql.DB
}

func NewPostgresTodo(db *sql.DB) *PostgresTodoRepository {
	return &PostgresTodoRepository{db: db}
}

func (r *PostgresTodoRepository) Create(ctx context.Context, todo model.Todo) (model.Todo, error) {
	query := `
		INSERT INTO todos (user_id, title, description, status, due_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, description, status, due_at, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query,
		todo.UserID, todo.Title, todo.Description, todo.Status, todo.DueAt,
	)

	return scanTodo(row)
}

func (r *PostgresTodoRepository) GetByID(ctx context.Context, userID, todoID string) (model.Todo, error) {
	query := `
		SELECT id, user_id, title, description, status, due_at, created_at, updated_at
		FROM todos
		WHERE id = $1 AND user_id = $2`

	row := r.db.QueryRowContext(ctx, query, todoID, userID)
	return scanTodo(row)
}

func (r *PostgresTodoRepository) Update(ctx context.Context, todo model.Todo) (model.Todo, error) {
	query := `
		UPDATE todos
		SET title = $1, description = $2, status = $3, due_at = $4, updated_at = now()
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, title, description, status, due_at, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, query,
		todo.Title, todo.Description, todo.Status, todo.DueAt, todo.ID, todo.UserID,
	)

	return scanTodo(row)
}

func (r *PostgresTodoRepository) Delete(ctx context.Context, userID, todoID string) error {
	query := `DELETE FROM todos WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, todoID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresTodoRepository) List(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Fetch one extra to determine if there's a next page
	fetchLimit := limit + 1

	args := []any{params.UserID}
	argIdx := 2

	query := `
		SELECT id, user_id, title, description, status, due_at, created_at, updated_at
		FROM todos
		WHERE user_id = $1`

	if params.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*params.Status))
		argIdx++
	}

	if params.Cursor != "" {
		query += fmt.Sprintf(" AND created_at < (SELECT created_at FROM todos WHERE id = $%d)", argIdx)
		args = append(args, params.Cursor)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, fetchLimit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return model.TodoListResult{}, fmt.Errorf("failed to list todos: %w", err)
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		todo, err := scanTodoFromRows(rows)
		if err != nil {
			return model.TodoListResult{}, err
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return model.TodoListResult{}, fmt.Errorf("failed to iterate todos: %w", err)
	}

	var nextCursor string
	if len(todos) > limit {
		nextCursor = todos[limit-1].ID
		todos = todos[:limit]
	}

	if todos == nil {
		todos = []model.Todo{}
	}

	return model.TodoListResult{
		Todos:      todos,
		NextCursor: nextCursor,
	}, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTodo(row scannable) (model.Todo, error) {
	var t model.Todo
	err := row.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description,
		&t.Status, &t.DueAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return model.Todo{}, fmt.Errorf("failed to scan todo: %w", err)
	}
	return t, nil
}

func scanTodoFromRows(rows *sql.Rows) (model.Todo, error) {
	var t model.Todo
	var dueAt sql.NullTime
	err := rows.Scan(
		&t.ID, &t.UserID, &t.Title, &t.Description,
		&t.Status, &dueAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return model.Todo{}, fmt.Errorf("failed to scan todo row: %w", err)
	}
	if dueAt.Valid {
		t.DueAt = &dueAt.Time
	}
	return t, nil
}

// ensure compile-time interface compliance
var _ TodoRepository = (*PostgresTodoRepository)(nil)
