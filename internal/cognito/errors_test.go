package cognito_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/cognito"
)

func TestLookupError_AllSentinels(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{cognito.ErrUserAlreadyExists, 409, "USER_ALREADY_EXISTS"},
		{cognito.ErrUserNotFound, 404, "USER_NOT_FOUND"},
		{cognito.ErrUserNotConfirmed, 403, "USER_NOT_CONFIRMED"},
		{cognito.ErrInvalidPassword, 400, "INVALID_PASSWORD"},
		{cognito.ErrInvalidCode, 400, "INVALID_CODE"},
		{cognito.ErrCodeExpired, 400, "CODE_EXPIRED"},
		{cognito.ErrTooManyRequests, 429, "TOO_MANY_REQUESTS"},
		{cognito.ErrNotAuthorized, 401, "NOT_AUTHORIZED"},
		{cognito.ErrLimitExceeded, 429, "LIMIT_EXCEEDED"},
		{cognito.ErrPasswordResetRequired, 403, "PASSWORD_RESET_REQUIRED"},
		{cognito.ErrInvalidParameter, 400, "INVALID_PARAMETER"},
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			info, ok := cognito.LookupError(tt.err)
			if !ok {
				t.Fatalf("expected LookupError to find %v", tt.err)
			}
			if info.Status != tt.wantStatus {
				t.Errorf("status: got %d, want %d", info.Status, tt.wantStatus)
			}
			if info.Code != tt.wantCode {
				t.Errorf("code: got %q, want %q", info.Code, tt.wantCode)
			}
		})
	}
}

func TestLookupError_WrappedError(t *testing.T) {
	wrapped := fmt.Errorf("something failed: %w", cognito.ErrUserNotFound)
	info, ok := cognito.LookupError(wrapped)
	if !ok {
		t.Fatal("expected LookupError to find wrapped error")
	}
	if info.Status != 404 {
		t.Errorf("status: got %d, want 404", info.Status)
	}
}

func TestLookupError_UnknownError(t *testing.T) {
	_, ok := cognito.LookupError(errors.New("unknown error"))
	if ok {
		t.Error("expected LookupError to return false for unknown error")
	}
}
