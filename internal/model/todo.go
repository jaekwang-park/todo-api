package model

import "time"

type TodoStatus string

const (
	TodoStatusPending   TodoStatus = "pending"
	TodoStatusCompleted TodoStatus = "completed"
)

func (s TodoStatus) IsValid() bool {
	return s == TodoStatusPending || s == TodoStatusCompleted
}

type Todo struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TodoStatus `json:"status"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TodoListParams struct {
	UserID string
	Status *TodoStatus
	Cursor string
	Limit  int
}

type TodoListResult struct {
	Todos      []Todo `json:"todos"`
	NextCursor string `json:"next_cursor,omitempty"`
}
