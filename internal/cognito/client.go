package cognito

import "context"

// Client defines the interface for Cognito authentication operations.
type Client interface {
	SignUp(ctx context.Context, input SignUpInput) (SignUpOutput, error)
	ConfirmSignUp(ctx context.Context, input ConfirmSignUpInput) error
	ResendConfirmationCode(ctx context.Context, input ResendCodeInput) error
	Login(ctx context.Context, input LoginInput) (AuthOutput, error)
	RefreshTokens(ctx context.Context, input RefreshInput) (AuthOutput, error)
	ForgotPassword(ctx context.Context, input ForgotPasswordInput) error
	ConfirmForgotPassword(ctx context.Context, input ConfirmForgotPasswordInput) error
	ChangePassword(ctx context.Context, input ChangePasswordInput) error
	GlobalSignOut(ctx context.Context, input GlobalSignOutInput) error
}

// SignUpInput contains the parameters for signing up a new user.
type SignUpInput struct {
	Email    string
	Password string
}

// SignUpOutput contains the result of a successful sign-up.
type SignUpOutput struct {
	UserSub       string
	Confirmed     bool
	CodeDelivery  string // e.g., "email"
}

// ConfirmSignUpInput contains the parameters for confirming a sign-up.
type ConfirmSignUpInput struct {
	Email string
	Code  string
}

// ResendCodeInput contains the parameters for resending a confirmation code.
type ResendCodeInput struct {
	Email string
}

// LoginInput contains the parameters for logging in a user.
type LoginInput struct {
	Email    string
	Password string
}

// AuthOutput contains tokens returned after successful authentication.
type AuthOutput struct {
	IDToken      string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int32
	TokenType    string
}

// RefreshInput contains the parameters for refreshing tokens.
type RefreshInput struct {
	Email        string
	RefreshToken string
}

// ForgotPasswordInput contains the parameters for initiating a password reset.
type ForgotPasswordInput struct {
	Email string
}

// ConfirmForgotPasswordInput contains the parameters for confirming a password reset.
type ConfirmForgotPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

// ChangePasswordInput contains the parameters for changing a password.
type ChangePasswordInput struct {
	AccessToken     string
	PreviousPassword string
	NewPassword      string
}

// GlobalSignOutInput contains the parameters for signing out globally.
type GlobalSignOutInput struct {
	AccessToken string
}
