package cognito

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/smithy-go"
)

// AWSClient implements Client using the AWS SDK v2.
type AWSClient struct {
	cip          *cip.Client
	clientID     string
	clientSecret string
}

// NewAWSClient creates a new AWSClient for the given region and app client.
func NewAWSClient(ctx context.Context, region, clientID, clientSecret string) (*AWSClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &AWSClient{
		cip:          cip.NewFromConfig(cfg),
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

func (c *AWSClient) secretHash(username string) *string {
	if c.clientSecret == "" {
		return nil
	}
	h := ComputeSecretHash(username, c.clientID, c.clientSecret)
	return &h
}

func (c *AWSClient) SignUp(ctx context.Context, input SignUpInput) (SignUpOutput, error) {
	out, err := c.cip.SignUp(ctx, &cip.SignUpInput{
		ClientId:   &c.clientID,
		SecretHash: c.secretHash(input.Email),
		Username:   &input.Email,
		Password:   &input.Password,
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: &input.Email},
		},
	})
	if err != nil {
		return SignUpOutput{}, mapAWSError(err)
	}
	delivery := ""
	if out.CodeDeliveryDetails != nil && out.CodeDeliveryDetails.DeliveryMedium != "" {
		delivery = string(out.CodeDeliveryDetails.DeliveryMedium)
	}
	return SignUpOutput{
		UserSub:      aws.ToString(out.UserSub),
		Confirmed:    out.UserConfirmed,
		CodeDelivery: delivery,
	}, nil
}

func (c *AWSClient) ConfirmSignUp(ctx context.Context, input ConfirmSignUpInput) error {
	_, err := c.cip.ConfirmSignUp(ctx, &cip.ConfirmSignUpInput{
		ClientId:         &c.clientID,
		SecretHash:       c.secretHash(input.Email),
		Username:         &input.Email,
		ConfirmationCode: &input.Code,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func (c *AWSClient) ResendConfirmationCode(ctx context.Context, input ResendCodeInput) error {
	_, err := c.cip.ResendConfirmationCode(ctx, &cip.ResendConfirmationCodeInput{
		ClientId:   &c.clientID,
		SecretHash: c.secretHash(input.Email),
		Username:   &input.Email,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func (c *AWSClient) Login(ctx context.Context, input LoginInput) (AuthOutput, error) {
	authParams := map[string]string{
		"USERNAME": input.Email,
		"PASSWORD": input.Password,
	}
	if h := c.secretHash(input.Email); h != nil {
		authParams["SECRET_HASH"] = *h
	}

	out, err := c.cip.InitiateAuth(ctx, &cip.InitiateAuthInput{
		ClientId:       &c.clientID,
		AuthFlow:       types.AuthFlowTypeUserPasswordAuth,
		AuthParameters: authParams,
	})
	if err != nil {
		return AuthOutput{}, mapAWSError(err)
	}
	if out.AuthenticationResult == nil {
		return AuthOutput{}, fmt.Errorf("unexpected nil authentication result")
	}
	return authOutputFromResult(out.AuthenticationResult), nil
}

func (c *AWSClient) RefreshTokens(ctx context.Context, input RefreshInput) (AuthOutput, error) {
	authParams := map[string]string{
		"REFRESH_TOKEN": input.RefreshToken,
	}
	if h := c.secretHash(input.Email); h != nil {
		authParams["SECRET_HASH"] = *h
	}

	out, err := c.cip.InitiateAuth(ctx, &cip.InitiateAuthInput{
		ClientId:       &c.clientID,
		AuthFlow:       types.AuthFlowTypeRefreshTokenAuth,
		AuthParameters: authParams,
	})
	if err != nil {
		return AuthOutput{}, mapAWSError(err)
	}
	if out.AuthenticationResult == nil {
		return AuthOutput{}, fmt.Errorf("unexpected nil authentication result")
	}
	return authOutputFromResult(out.AuthenticationResult), nil
}

func (c *AWSClient) ForgotPassword(ctx context.Context, input ForgotPasswordInput) error {
	_, err := c.cip.ForgotPassword(ctx, &cip.ForgotPasswordInput{
		ClientId:   &c.clientID,
		SecretHash: c.secretHash(input.Email),
		Username:   &input.Email,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func (c *AWSClient) ConfirmForgotPassword(ctx context.Context, input ConfirmForgotPasswordInput) error {
	_, err := c.cip.ConfirmForgotPassword(ctx, &cip.ConfirmForgotPasswordInput{
		ClientId:         &c.clientID,
		SecretHash:       c.secretHash(input.Email),
		Username:         &input.Email,
		ConfirmationCode: &input.Code,
		Password:         &input.NewPassword,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func (c *AWSClient) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	_, err := c.cip.ChangePassword(ctx, &cip.ChangePasswordInput{
		AccessToken:      &input.AccessToken,
		PreviousPassword: &input.PreviousPassword,
		ProposedPassword: &input.NewPassword,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func (c *AWSClient) GlobalSignOut(ctx context.Context, input GlobalSignOutInput) error {
	_, err := c.cip.GlobalSignOut(ctx, &cip.GlobalSignOutInput{
		AccessToken: &input.AccessToken,
	})
	if err != nil {
		return mapAWSError(err)
	}
	return nil
}

func authOutputFromResult(r *types.AuthenticationResultType) AuthOutput {
	return AuthOutput{
		IDToken:      aws.ToString(r.IdToken),
		AccessToken:  aws.ToString(r.AccessToken),
		RefreshToken: aws.ToString(r.RefreshToken),
		ExpiresIn:    r.ExpiresIn,
		TokenType:    aws.ToString(r.TokenType),
	}
}

// mapAWSError converts AWS SDK errors to cognito sentinel errors.
func mapAWSError(err error) error {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("cognito: %w", err)
	}

	switch apiErr.ErrorCode() {
	case "UsernameExistsException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrUserAlreadyExists)
	case "UserNotFoundException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrUserNotFound)
	case "UserNotConfirmedException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrUserNotConfirmed)
	case "InvalidPasswordException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrInvalidPassword)
	case "CodeMismatchException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrInvalidCode)
	case "ExpiredCodeException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrCodeExpired)
	case "TooManyRequestsException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrTooManyRequests)
	case "NotAuthorizedException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrNotAuthorized)
	case "LimitExceededException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrLimitExceeded)
	case "PasswordResetRequiredException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrPasswordResetRequired)
	case "InvalidParameterException":
		return fmt.Errorf("%s: %w", apiErr.ErrorMessage(), ErrInvalidParameter)
	default:
		return fmt.Errorf("cognito %s: %w", apiErr.ErrorCode(), err)
	}
}

// Compile-time check: AWSClient implements Client.
var _ Client = (*AWSClient)(nil)
