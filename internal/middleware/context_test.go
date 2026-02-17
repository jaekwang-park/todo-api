package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/middleware"
)

func TestSetAndGetUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	// Before setting — should return empty
	if got := middleware.GetUserID(req); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// After setting
	ctx := middleware.SetUserID(req.Context(), "user-abc")
	req = req.WithContext(ctx)

	if got := middleware.GetUserID(req); got != "user-abc" {
		t.Errorf("expected user-abc, got %q", got)
	}
}
