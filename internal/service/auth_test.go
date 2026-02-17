package service_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/cognito"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// --- Mock Cognito Client ---

type mockCognitoClient struct {
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

func (m *mockCognitoClient) SignUp(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error) {
	return m.signUpFn(ctx, input)
}
func (m *mockCognitoClient) ConfirmSignUp(ctx context.Context, input cognito.ConfirmSignUpInput) error {
	return m.confirmSignUpFn(ctx, input)
}
func (m *mockCognitoClient) ResendConfirmationCode(ctx context.Context, input cognito.ResendCodeInput) error {
	return m.resendConfirmationCodeFn(ctx, input)
}
func (m *mockCognitoClient) Login(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error) {
	return m.loginFn(ctx, input)
}
func (m *mockCognitoClient) RefreshTokens(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error) {
	return m.refreshTokensFn(ctx, input)
}
func (m *mockCognitoClient) ForgotPassword(ctx context.Context, input cognito.ForgotPasswordInput) error {
	return m.forgotPasswordFn(ctx, input)
}
func (m *mockCognitoClient) ConfirmForgotPassword(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error {
	return m.confirmForgotPasswordFn(ctx, input)
}
func (m *mockCognitoClient) ChangePassword(ctx context.Context, input cognito.ChangePasswordInput) error {
	return m.changePasswordFn(ctx, input)
}
func (m *mockCognitoClient) GlobalSignOut(ctx context.Context, input cognito.GlobalSignOutInput) error {
	return m.globalSignOutFn(ctx, input)
}

// --- Mock User Repository ---

type mockUserRepo struct {
	getOrCreateFn     func(ctx context.Context, cognitoSub, email string) (model.User, error)
	getByCognitoSubFn func(ctx context.Context, cognitoSub string) (model.User, error)
	updateFn          func(ctx context.Context, user model.User) (model.User, error)
}

func (m *mockUserRepo) GetOrCreate(ctx context.Context, cognitoSub, email string) (model.User, error) {
	return m.getOrCreateFn(ctx, cognitoSub, email)
}
func (m *mockUserRepo) GetByCognitoSub(ctx context.Context, cognitoSub string) (model.User, error) {
	return m.getByCognitoSubFn(ctx, cognitoSub)
}
func (m *mockUserRepo) Update(ctx context.Context, user model.User) (model.User, error) {
	return m.updateFn(ctx, user)
}

// fakeIDToken creates a JWT-like string with a base64url-encoded payload
// containing the given sub and email claims.
func fakeIDToken(sub, email string) string {
	header := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9"
	payloadJSON := `{"sub":"` + sub + `","email":"` + email + `"}`
	payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	return header + "." + payload + ".fakesig"
}

func TestAuthService_SignUp(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		mockOut   cognito.SignUpOutput
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:     "success",
			email:    "test@example.com",
			password: "Password1!",
			mockOut: cognito.SignUpOutput{
				UserSub:      "sub-123",
				Confirmed:    false,
				CodeDelivery: "EMAIL",
			},
		},
		{
			name:     "empty email",
			email:    "",
			password: "Password1!",
			wantErr:  true,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			wantErr:  true,
		},
		{
			name:      "cognito error: user already exists",
			email:     "test@example.com",
			password:  "Password1!",
			mockErr:   cognito.ErrUserAlreadyExists,
			wantErr:   true,
			wantErrIs: cognito.ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				signUpFn: func(ctx context.Context, input cognito.SignUpInput) (cognito.SignUpOutput, error) {
					if tt.mockErr != nil {
						return cognito.SignUpOutput{}, tt.mockErr
					}
					return tt.mockOut, nil
				},
			}
			svc := service.NewAuthService(mock, nil)

			out, err := svc.SignUp(context.Background(), service.SignUpInput{
				Email:    tt.email,
				Password: tt.password,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.UserSub != tt.mockOut.UserSub {
				t.Errorf("UserSub: got %q, want %q", out.UserSub, tt.mockOut.UserSub)
			}
		})
	}
}

func TestAuthService_ConfirmSignUp(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		code      string
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:  "success",
			email: "test@example.com",
			code:  "123456",
		},
		{
			name:    "empty email",
			email:   "",
			code:    "123456",
			wantErr: true,
		},
		{
			name:    "empty code",
			email:   "test@example.com",
			code:    "",
			wantErr: true,
		},
		{
			name:      "invalid code",
			email:     "test@example.com",
			code:      "wrong",
			mockErr:   cognito.ErrInvalidCode,
			wantErr:   true,
			wantErrIs: cognito.ErrInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				confirmSignUpFn: func(ctx context.Context, input cognito.ConfirmSignUpInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.ConfirmSignUp(context.Background(), service.ConfirmSignUpInput{
				Email: tt.email,
				Code:  tt.code,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_ResendCode(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:  "success",
			email: "test@example.com",
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:      "user not found",
			email:     "test@example.com",
			mockErr:   cognito.ErrUserNotFound,
			wantErr:   true,
			wantErrIs: cognito.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				resendConfirmationCodeFn: func(ctx context.Context, input cognito.ResendCodeInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.ResendCode(context.Background(), service.ResendCodeInput{
				Email: tt.email,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		mockAuthOut   cognito.AuthOutput
		mockErr       error
		mockUserErr   error
		wantErr       bool
		wantErrIs     error
		wantTokenType string
	}{
		{
			name:     "success",
			email:    "test@example.com",
			password: "Password1!",
			mockAuthOut: cognito.AuthOutput{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			},
			wantTokenType: "Bearer",
		},
		{
			name:     "empty email",
			email:    "",
			password: "Password1!",
			wantErr:  true,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			password: "",
			wantErr:  true,
		},
		{
			name:      "wrong credentials",
			email:     "test@example.com",
			password:  "wrong",
			mockErr:   cognito.ErrNotAuthorized,
			wantErr:   true,
			wantErrIs: cognito.ErrNotAuthorized,
		},
		{
			name:     "user repo error",
			email:    "test@example.com",
			password: "Password1!",
			mockAuthOut: cognito.AuthOutput{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			},
			mockUserErr: errors.New("db error"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := "cognito-sub-123"
			authOut := tt.mockAuthOut
			if authOut.IDToken == "" && tt.mockErr == nil {
				authOut.IDToken = fakeIDToken(sub, tt.email)
			}

			mock := &mockCognitoClient{
				loginFn: func(ctx context.Context, input cognito.LoginInput) (cognito.AuthOutput, error) {
					if tt.mockErr != nil {
						return cognito.AuthOutput{}, tt.mockErr
					}
					return authOut, nil
				},
			}

			userRepo := &mockUserRepo{
				getOrCreateFn: func(ctx context.Context, cognitoSub, email string) (model.User, error) {
					if tt.mockUserErr != nil {
						return model.User{}, tt.mockUserErr
					}
					return model.User{ID: "user-1", CognitoSub: cognitoSub, Email: email}, nil
				},
			}

			svc := service.NewAuthService(mock, userRepo)

			out, err := svc.Login(context.Background(), service.LoginInput{
				Email:    tt.email,
				Password: tt.password,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.AccessToken != authOut.AccessToken {
				t.Errorf("AccessToken: got %q, want %q", out.AccessToken, authOut.AccessToken)
			}
			if out.TokenType != tt.wantTokenType {
				t.Errorf("TokenType: got %q, want %q", out.TokenType, tt.wantTokenType)
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		token     string
		mockOut   cognito.AuthOutput
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:  "success",
			email: "test@example.com",
			token: "refresh-token-abc",
			mockOut: cognito.AuthOutput{
				IDToken:     "new-id-token",
				AccessToken: "new-access-token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			},
		},
		{
			name:    "empty email",
			email:   "",
			token:   "refresh-token-abc",
			wantErr: true,
		},
		{
			name:    "empty refresh token",
			email:   "test@example.com",
			token:   "",
			wantErr: true,
		},
		{
			name:      "invalid refresh token",
			email:     "test@example.com",
			token:     "invalid",
			mockErr:   cognito.ErrNotAuthorized,
			wantErr:   true,
			wantErrIs: cognito.ErrNotAuthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				refreshTokensFn: func(ctx context.Context, input cognito.RefreshInput) (cognito.AuthOutput, error) {
					if tt.mockErr != nil {
						return cognito.AuthOutput{}, tt.mockErr
					}
					return tt.mockOut, nil
				},
			}
			svc := service.NewAuthService(mock, nil)

			out, err := svc.Refresh(context.Background(), service.RefreshInput{
				Email:        tt.email,
				RefreshToken: tt.token,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.AccessToken != tt.mockOut.AccessToken {
				t.Errorf("AccessToken: got %q, want %q", out.AccessToken, tt.mockOut.AccessToken)
			}
		})
	}
}

func TestAuthService_ForgotPassword(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:  "success",
			email: "test@example.com",
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:      "user not found",
			email:     "test@example.com",
			mockErr:   cognito.ErrUserNotFound,
			wantErr:   true,
			wantErrIs: cognito.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				forgotPasswordFn: func(ctx context.Context, input cognito.ForgotPasswordInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.ForgotPassword(context.Background(), service.ForgotPasswordInput{
				Email: tt.email,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_ConfirmForgotPassword(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		code      string
		password  string
		mockErr   error
		wantErr   bool
		wantErrIs error
	}{
		{
			name:     "success",
			email:    "test@example.com",
			code:     "123456",
			password: "NewPassword1!",
		},
		{
			name:     "empty email",
			email:    "",
			code:     "123456",
			password: "NewPassword1!",
			wantErr:  true,
		},
		{
			name:     "empty code",
			email:    "test@example.com",
			code:     "",
			password: "NewPassword1!",
			wantErr:  true,
		},
		{
			name:     "empty password",
			email:    "test@example.com",
			code:     "123456",
			password: "",
			wantErr:  true,
		},
		{
			name:      "invalid code",
			email:     "test@example.com",
			code:      "wrong",
			password:  "NewPassword1!",
			mockErr:   cognito.ErrInvalidCode,
			wantErr:   true,
			wantErrIs: cognito.ErrInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				confirmForgotPasswordFn: func(ctx context.Context, input cognito.ConfirmForgotPasswordInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.ConfirmForgotPassword(context.Background(), service.ConfirmForgotPasswordInput{
				Email:       tt.email,
				Code:        tt.code,
				NewPassword: tt.password,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	tests := []struct {
		name         string
		accessToken  string
		prevPassword string
		newPassword  string
		mockErr      error
		wantErr      bool
		wantErrIs    error
	}{
		{
			name:         "success",
			accessToken:  "access-token",
			prevPassword: "OldPassword1!",
			newPassword:  "NewPassword1!",
		},
		{
			name:         "empty access token",
			accessToken:  "",
			prevPassword: "OldPassword1!",
			newPassword:  "NewPassword1!",
			wantErr:      true,
		},
		{
			name:         "empty previous password",
			accessToken:  "access-token",
			prevPassword: "",
			newPassword:  "NewPassword1!",
			wantErr:      true,
		},
		{
			name:         "empty new password",
			accessToken:  "access-token",
			prevPassword: "OldPassword1!",
			newPassword:  "",
			wantErr:      true,
		},
		{
			name:         "wrong old password",
			accessToken:  "access-token",
			prevPassword: "wrong",
			newPassword:  "NewPassword1!",
			mockErr:      cognito.ErrNotAuthorized,
			wantErr:      true,
			wantErrIs:    cognito.ErrNotAuthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				changePasswordFn: func(ctx context.Context, input cognito.ChangePasswordInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.ChangePassword(context.Background(), service.ChangePasswordInput{
				AccessToken:      tt.accessToken,
				PreviousPassword: tt.prevPassword,
				NewPassword:      tt.newPassword,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name        string
		accessToken string
		mockErr     error
		wantErr     bool
		wantErrIs   error
	}{
		{
			name:        "success",
			accessToken: "access-token",
		},
		{
			name:        "empty access token",
			accessToken: "",
			wantErr:     true,
		},
		{
			name:        "not authorized",
			accessToken: "invalid",
			mockErr:     cognito.ErrNotAuthorized,
			wantErr:     true,
			wantErrIs:   cognito.ErrNotAuthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCognitoClient{
				globalSignOutFn: func(ctx context.Context, input cognito.GlobalSignOutInput) error {
					return tt.mockErr
				},
			}
			svc := service.NewAuthService(mock, nil)

			err := svc.Logout(context.Background(), service.LogoutInput{
				AccessToken: tt.accessToken,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
