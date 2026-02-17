package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jaekwang-park/todo-api/internal/cognito"
	"github.com/jaekwang-park/todo-api/internal/repository"
)

// AuthService handles authentication-related business logic.
type AuthService struct {
	cognitoClient cognito.Client
	userRepo      repository.UserRepository
}

// NewAuthService creates a new AuthService.
func NewAuthService(cognitoClient cognito.Client, userRepo repository.UserRepository) *AuthService {
	return &AuthService{
		cognitoClient: cognitoClient,
		userRepo:      userRepo,
	}
}

// --- Input/Output types ---

type SignUpInput struct {
	Email    string
	Password string
}

type SignUpOutput struct {
	UserSub      string `json:"user_sub"`
	Confirmed    bool   `json:"confirmed"`
	CodeDelivery string `json:"code_delivery"`
}

type ConfirmSignUpInput struct {
	Email string
	Code  string
}

type ResendCodeInput struct {
	Email string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int32  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type RefreshInput struct {
	Email        string
	RefreshToken string
}

type RefreshOutput struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int32  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type ForgotPasswordInput struct {
	Email string
}

type ConfirmForgotPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

type ChangePasswordInput struct {
	AccessToken      string
	PreviousPassword string
	NewPassword      string
}

type LogoutInput struct {
	AccessToken string
}

// --- Service methods ---

func (s *AuthService) SignUp(ctx context.Context, input SignUpInput) (SignUpOutput, error) {
	if input.Email == "" {
		return SignUpOutput{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.Password == "" {
		return SignUpOutput{}, fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	out, err := s.cognitoClient.SignUp(ctx, cognito.SignUpInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return SignUpOutput{}, err
	}

	return SignUpOutput{
		UserSub:      out.UserSub,
		Confirmed:    out.Confirmed,
		CodeDelivery: out.CodeDelivery,
	}, nil
}

func (s *AuthService) ConfirmSignUp(ctx context.Context, input ConfirmSignUpInput) error {
	if input.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidInput)
	}

	return s.cognitoClient.ConfirmSignUp(ctx, cognito.ConfirmSignUpInput{
		Email: input.Email,
		Code:  input.Code,
	})
}

func (s *AuthService) ResendCode(ctx context.Context, input ResendCodeInput) error {
	if input.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	return s.cognitoClient.ResendConfirmationCode(ctx, cognito.ResendCodeInput{
		Email: input.Email,
	})
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (LoginOutput, error) {
	if input.Email == "" {
		return LoginOutput{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.Password == "" {
		return LoginOutput{}, fmt.Errorf("%w: password is required", ErrInvalidInput)
	}

	authOut, err := s.cognitoClient.Login(ctx, cognito.LoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return LoginOutput{}, err
	}

	// Extract sub from ID token payload (no signature verification needed — just issued by Cognito)
	sub, err := extractSub(authOut.IDToken)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("failed to extract sub from id token: %w", err)
	}

	// Create or update user in DB
	if _, err := s.userRepo.GetOrCreate(ctx, sub, input.Email); err != nil {
		return LoginOutput{}, fmt.Errorf("failed to get or create user: %w", err)
	}

	return LoginOutput{
		IDToken:      authOut.IDToken,
		AccessToken:  authOut.AccessToken,
		RefreshToken: authOut.RefreshToken,
		ExpiresIn:    authOut.ExpiresIn,
		TokenType:    authOut.TokenType,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (RefreshOutput, error) {
	if input.Email == "" {
		return RefreshOutput{}, fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.RefreshToken == "" {
		return RefreshOutput{}, fmt.Errorf("%w: refresh_token is required", ErrInvalidInput)
	}

	authOut, err := s.cognitoClient.RefreshTokens(ctx, cognito.RefreshInput{
		Email:        input.Email,
		RefreshToken: input.RefreshToken,
	})
	if err != nil {
		return RefreshOutput{}, err
	}

	return RefreshOutput{
		IDToken:     authOut.IDToken,
		AccessToken: authOut.AccessToken,
		ExpiresIn:   authOut.ExpiresIn,
		TokenType:   authOut.TokenType,
	}, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, input ForgotPasswordInput) error {
	if input.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}

	return s.cognitoClient.ForgotPassword(ctx, cognito.ForgotPasswordInput{
		Email: input.Email,
	})
}

func (s *AuthService) ConfirmForgotPassword(ctx context.Context, input ConfirmForgotPasswordInput) error {
	if input.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if input.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if input.NewPassword == "" {
		return fmt.Errorf("%w: new_password is required", ErrInvalidInput)
	}

	return s.cognitoClient.ConfirmForgotPassword(ctx, cognito.ConfirmForgotPasswordInput{
		Email:       input.Email,
		Code:        input.Code,
		NewPassword: input.NewPassword,
	})
}

func (s *AuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	if input.AccessToken == "" {
		return fmt.Errorf("%w: access_token is required", ErrInvalidInput)
	}
	if input.PreviousPassword == "" {
		return fmt.Errorf("%w: previous_password is required", ErrInvalidInput)
	}
	if input.NewPassword == "" {
		return fmt.Errorf("%w: new_password is required", ErrInvalidInput)
	}

	return s.cognitoClient.ChangePassword(ctx, cognito.ChangePasswordInput{
		AccessToken:      input.AccessToken,
		PreviousPassword: input.PreviousPassword,
		NewPassword:      input.NewPassword,
	})
}

func (s *AuthService) Logout(ctx context.Context, input LogoutInput) error {
	if input.AccessToken == "" {
		return fmt.Errorf("%w: access_token is required", ErrInvalidInput)
	}

	return s.cognitoClient.GlobalSignOut(ctx, cognito.GlobalSignOutInput{
		AccessToken: input.AccessToken,
	})
}

// extractSub decodes the JWT payload (without verifying signature) and extracts the "sub" claim.
func extractSub(idToken string) (string, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}
	if claims.Sub == "" {
		return "", fmt.Errorf("sub claim not found in JWT")
	}

	return claims.Sub, nil
}
