package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/cognito"
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

// stubCognitoClient for router tests — all methods return errors (not exercised)
type stubCognitoClient struct{}

func (s *stubCognitoClient) SignUp(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error) {
	return cognito.SignUpOutput{}, fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) ConfirmSignUp(ctx context.Context, input cognito.ConfirmSignUpInput) error {
	return fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) ResendConfirmationCode(ctx context.Context, input cognito.ResendCodeInput) error {
	return fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) Login(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error) {
	return cognito.AuthOutput{}, fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) RefreshTokens(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error) {
	return cognito.AuthOutput{}, fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) ForgotPassword(ctx context.Context, input cognito.ForgotPasswordInput) error {
	return fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) ConfirmForgotPassword(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error {
	return fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) ChangePassword(ctx context.Context, input cognito.ChangePasswordInput) error {
	return fmt.Errorf("not implemented")
}
func (s *stubCognitoClient) GlobalSignOut(ctx context.Context, input cognito.GlobalSignOutInput) error {
	return fmt.Errorf("not implemented")
}

func newTestTodoSvc() *service.TodoService {
	return service.NewTodoService(&mockTodoRepo{})
}

func newTestAuthSvc() *service.AuthService {
	return service.NewAuthService(&stubCognitoClient{}, nil)
}

func TestRouter_HealthEndpoint(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc(), newTestAuthSvc())

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
	router := todohttp.NewRouter(newTestTodoSvc(), newTestAuthSvc())

	// Set user ID in context to simulate auth middleware
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Router itself doesn't enforce auth — that's the middleware's job
	// Just verify the route is registered (200, not 404)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestRouter_AuthEndpointRegistered(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc(), newTestAuthSvc())

	// Auth signup with empty body → should get a JSON error (not 404)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// We expect a non-404 response (route is registered)
	if w.Code == http.StatusNotFound {
		t.Errorf("expected auth route to be registered, got 404")
	}
}

func TestRouter_UnknownRoute(t *testing.T) {
	router := todohttp.NewRouter(newTestTodoSvc(), newTestAuthSvc())

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
