package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/cognito"
	"github.com/jaekwang-park/todo-api/internal/http/handler"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// --- Mock Cognito Client ---

type mockAuthCognitoClient struct {
	signUpFn                 func(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error)
	confirmSignUpFn          func(ctx context.Context, input cognito.ConfirmSignUpInput) error
	resendConfirmationCodeFn func(ctx context.Context, input cognito.ResendCodeInput) error
	loginFn                  func(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error)
	refreshTokensFn          func(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error)
	forgotPasswordFn         func(ctx context.Context, input cognito.ForgotPasswordInput) error
	confirmForgotPasswordFn  func(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error
	changePasswordFn         func(ctx context.Context, input cognito.ChangePasswordInput) error
	globalSignOutFn          func(ctx context.Context, input cognito.GlobalSignOutInput) error
}

func (m *mockAuthCognitoClient) SignUp(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error) {
	return m.signUpFn(ctx, input)
}
func (m *mockAuthCognitoClient) ConfirmSignUp(ctx context.Context, input cognito.ConfirmSignUpInput) error {
	return m.confirmSignUpFn(ctx, input)
}
func (m *mockAuthCognitoClient) ResendConfirmationCode(ctx context.Context, input cognito.ResendCodeInput) error {
	return m.resendConfirmationCodeFn(ctx, input)
}
func (m *mockAuthCognitoClient) Login(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error) {
	return m.loginFn(ctx, input)
}
func (m *mockAuthCognitoClient) RefreshTokens(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error) {
	return m.refreshTokensFn(ctx, input)
}
func (m *mockAuthCognitoClient) ForgotPassword(ctx context.Context, input cognito.ForgotPasswordInput) error {
	return m.forgotPasswordFn(ctx, input)
}
func (m *mockAuthCognitoClient) ConfirmForgotPassword(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error {
	return m.confirmForgotPasswordFn(ctx, input)
}
func (m *mockAuthCognitoClient) ChangePassword(ctx context.Context, input cognito.ChangePasswordInput) error {
	return m.changePasswordFn(ctx, input)
}
func (m *mockAuthCognitoClient) GlobalSignOut(ctx context.Context, input cognito.GlobalSignOutInput) error {
	return m.globalSignOutFn(ctx, input)
}

// --- Mock User Repository ---

type mockAuthUserRepo struct {
	getOrCreateFn     func(ctx context.Context, cognitoSub, email string) (model.User, error)
	getByCognitoSubFn func(ctx context.Context, cognitoSub string) (model.User, error)
	updateFn          func(ctx context.Context, user model.User) (model.User, error)
}

func (m *mockAuthUserRepo) GetOrCreate(ctx context.Context, cognitoSub, email string) (model.User, error) {
	return m.getOrCreateFn(ctx, cognitoSub, email)
}
func (m *mockAuthUserRepo) GetByCognitoSub(ctx context.Context, cognitoSub string) (model.User, error) {
	return m.getByCognitoSubFn(ctx, cognitoSub)
}
func (m *mockAuthUserRepo) Update(ctx context.Context, user model.User) (model.User, error) {
	return m.updateFn(ctx, user)
}

func fakeIDTokenForHandler(sub, email string) string {
	header := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
	payloadJSON := `{"sub":"` + sub + `","email":"` + email + `"}`
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	return header + "." + payload + ".fakesig"
}

func newAuthHandler(cognitoClient cognito.Client, userRepo *mockAuthUserRepo) *handler.AuthHandler {
	svc := service.NewAuthService(cognitoClient, userRepo)
	return handler.NewAuthHandler(svc)
}

func TestAuthHandler_SignUp(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com","password":"Password1!"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_JSON",
		},
		{
			name:       "missing email",
			body:       `{"email":"","password":"Password1!"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "user already exists",
			body:       `{"email":"test@example.com","password":"Password1!"}`,
			mockErr:    cognito.ErrUserAlreadyExists,
			wantStatus: http.StatusConflict,
			wantCode:   "USER_ALREADY_EXISTS",
		},
		{
			name:       "method not allowed",
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				signUpFn: func(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error) {
					if tt.mockErr != nil {
						return cognito.SignUpOutput{}, tt.mockErr
					}
					return cognito.SignUpOutput{UserSub: "sub-123", CodeDelivery: "EMAIL"}, nil
				},
			}

			h := newAuthHandler(mock, nil)

			var method string
			if tt.name == "method not allowed" {
				method = http.MethodGet
			} else {
				method = http.MethodPost
			}

			req := httptest.NewRequest(method, "/api/v1/auth/signup", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantCode != "" {
				var resp handler.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err == nil {
					if resp.Error.Code != tt.wantCode {
						t.Errorf("expected error code %q, got %q", tt.wantCode, resp.Error.Code)
					}
				}
			}
		})
	}
}

func TestAuthHandler_ConfirmSignUp(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com","code":"123456"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid code",
			body:       `{"email":"test@example.com","code":"wrong"}`,
			mockErr:    cognito.ErrInvalidCode,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				confirmSignUpFn: func(ctx context.Context, input cognito.ConfirmSignUpInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/confirm-signup", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_ResendCode(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user not found",
			body:       `{"email":"test@example.com"}`,
			mockErr:    cognito.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				resendConfirmationCodeFn: func(ctx context.Context, input cognito.ResendCodeInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-code", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com","password":"Password1!"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong credentials",
			body:       `{"email":"test@example.com","password":"wrong"}`,
			mockErr:    cognito.ErrNotAuthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not confirmed",
			body:       `{"email":"test@example.com","password":"Password1!"}`,
			mockErr:    cognito.ErrUserNotConfirmed,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := "cognito-sub-123"
			idToken := fakeIDTokenForHandler(sub, "test@example.com")

			mock := &mockAuthCognitoClient{
				loginFn: func(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error) {
					if tt.mockErr != nil {
						return cognito.AuthOutput{}, tt.mockErr
					}
					return cognito.AuthOutput{
						IDToken:      idToken,
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
						TokenType:    "Bearer",
					}, nil
				},
			}

			userRepo := &mockAuthUserRepo{
				getOrCreateFn: func(ctx context.Context, cognitoSub, email string) (model.User, error) {
					return model.User{ID: "user-1", CognitoSub: cognitoSub, Email: email}, nil
				},
			}

			h := newAuthHandler(mock, userRepo)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com","refresh_token":"refresh-abc"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid refresh token",
			body:       `{"email":"test@example.com","refresh_token":"invalid"}`,
			mockErr:    cognito.ErrNotAuthorized,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				refreshTokensFn: func(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error) {
					if tt.mockErr != nil {
						return cognito.AuthOutput{}, tt.mockErr
					}
					return cognito.AuthOutput{
						IDToken:     "new-id-token",
						AccessToken: "new-access-token",
						ExpiresIn:   3600,
						TokenType:   "Bearer",
					}, nil
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user not found",
			body:       `{"email":"test@example.com"}`,
			mockErr:    cognito.ErrUserNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				forgotPasswordFn: func(ctx context.Context, input cognito.ForgotPasswordInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_ConfirmForgotPassword(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"email":"test@example.com","code":"123456","new_password":"NewPass1!"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "expired code",
			body:       `{"email":"test@example.com","code":"old","new_password":"NewPass1!"}`,
			mockErr:    cognito.ErrCodeExpired,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				confirmForgotPasswordFn: func(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/confirm-forgot-password", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_ChangePassword(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"access_token":"tok","previous_password":"Old1!","new_password":"New1!"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "not authorized",
			body:       `{"access_token":"tok","previous_password":"wrong","new_password":"New1!"}`,
			mockErr:    cognito.ErrNotAuthorized,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				changePasswordFn: func(ctx context.Context, input cognito.ChangePasswordInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockErr    error
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"access_token":"tok"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "not authorized",
			body:       `{"access_token":"invalid"}`,
			mockErr:    cognito.ErrNotAuthorized,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAuthCognitoClient{
				globalSignOutFn: func(ctx context.Context, input cognito.GlobalSignOutInput) error {
					return tt.mockErr
				},
			}

			h := newAuthHandler(mock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_UnknownRoute(t *testing.T) {
	mock := &mockAuthCognitoClient{}
	h := newAuthHandler(mock, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/unknown", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
