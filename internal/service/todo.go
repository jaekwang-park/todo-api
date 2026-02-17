package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/repository"
)

// parseDueAt parses an RFC3339 string into *time.Time.
// Returns nil if input is nil.
func parseDueAt(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid due_at format, expected RFC3339", ErrInvalidInput)
	}
	return &t, nil
}

type CreateTodoInput struct {
	Title       string
	Description string
	DueAt       *string // RFC3339 string, parsed in handler
}

type UpdateTodoInput struct {
	Title       *string
	Description *string
	DueAt       *string
}

type TodoService struct {
	repo repository.TodoRepository
}

func NewTodoService(repo repository.TodoRepository) *TodoService {
	return &TodoService{repo: repo}
}

func (s *TodoService) Create(ctx context.Context, userID string, input CreateTodoInput) (model.Todo, error) {
	if input.Title == "" {
		return model.Todo{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	dueAt, err := parseDueAt(input.DueAt)
	if err != nil {
		return model.Todo{}, err
	}

	todo := model.Todo{
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		Status:      model.TodoStatusPending,
		DueAt:       dueAt,
	}

	created, err := s.repo.Create(ctx, todo)
	if err != nil {
		return model.Todo{}, fmt.Errorf("failed to create todo: %w", err)
	}

	return created, nil
}

func (s *TodoService) GetByID(ctx context.Context, userID, todoID string) (model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, userID, todoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Todo{}, ErrNotFound
		}
		return model.Todo{}, fmt.Errorf("failed to get todo: %w", err)
	}
	return todo, nil
}

func (s *TodoService) Update(ctx context.Context, userID, todoID string, input UpdateTodoInput) (model.Todo, error) {
	existing, err := s.repo.GetByID(ctx, userID, todoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Todo{}, ErrNotFound
		}
		return model.Todo{}, fmt.Errorf("failed to get todo for update: %w", err)
	}

	if input.Title != nil {
		if *input.Title == "" {
			return model.Todo{}, fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
		}
		existing.Title = *input.Title
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.DueAt != nil {
		dueAt, err := parseDueAt(input.DueAt)
		if err != nil {
			return model.Todo{}, err
		}
		existing.DueAt = dueAt
	}

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return model.Todo{}, fmt.Errorf("failed to update todo: %w", err)
	}

	return updated, nil
}

func (s *TodoService) Delete(ctx context.Context, userID, todoID string) error {
	err := s.repo.Delete(ctx, userID, todoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete todo: %w", err)
	}
	return nil
}

func (s *TodoService) UpdateStatus(ctx context.Context, userID, todoID string, status model.TodoStatus) (model.Todo, error) {
	if !status.IsValid() {
		return model.Todo{}, fmt.Errorf("%w: invalid status %q", ErrInvalidInput, status)
	}

	existing, err := s.repo.GetByID(ctx, userID, todoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Todo{}, ErrNotFound
		}
		return model.Todo{}, fmt.Errorf("failed to get todo for status update: %w", err)
	}

	existing.Status = status

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return model.Todo{}, fmt.Errorf("failed to update todo status: %w", err)
	}

	return updated, nil
}

func (s *TodoService) List(ctx context.Context, params model.TodoListParams) (model.TodoListResult, error) {
	result, err := s.repo.List(ctx, params)
	if err != nil {
		return model.TodoListResult{}, fmt.Errorf("failed to list todos: %w", err)
	}
	return result, nil
}
