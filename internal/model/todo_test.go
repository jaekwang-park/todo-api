package model_test

import (
	"testing"

	"github.com/jaekwang-park/todo-api/internal/model"
)

func TestTodoStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status model.TodoStatus
		want   bool
	}{
		{"pending", model.TodoStatusPending, true},
		{"completed", model.TodoStatusCompleted, true},
		{"empty", model.TodoStatus(""), false},
		{"invalid", model.TodoStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("TodoStatus(%q).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
