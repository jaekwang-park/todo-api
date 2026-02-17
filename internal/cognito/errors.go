package cognito

import "errors"

// Sentinel errors for Cognito operations.
var (
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserNotConfirmed    = errors.New("user not confirmed")
	ErrInvalidPassword     = errors.New("invalid password")
	ErrInvalidCode         = errors.New("invalid code")
	ErrCodeExpired         = errors.New("code expired")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrNotAuthorized       = errors.New("not authorized")
	ErrLimitExceeded       = errors.New("limit exceeded")
	ErrPasswordResetRequired = errors.New("password reset required")
	ErrInvalidParameter    = errors.New("invalid parameter")
)

// ErrorInfo maps a sentinel error to its HTTP status and error code.
type ErrorInfo struct {
	Status int
	Code   string
}

// errorMap maps sentinel errors to their HTTP status codes and error codes.
var errorMap = map[error]ErrorInfo{
	ErrUserAlreadyExists:   {Status: 409, Code: "USER_ALREADY_EXISTS"},
	ErrUserNotFound:        {Status: 404, Code: "USER_NOT_FOUND"},
	ErrUserNotConfirmed:    {Status: 403, Code: "USER_NOT_CONFIRMED"},
	ErrInvalidPassword:     {Status: 400, Code: "INVALID_PASSWORD"},
	ErrInvalidCode:         {Status: 400, Code: "INVALID_CODE"},
	ErrCodeExpired:         {Status: 400, Code: "CODE_EXPIRED"},
	ErrTooManyRequests:     {Status: 429, Code: "TOO_MANY_REQUESTS"},
	ErrNotAuthorized:       {Status: 401, Code: "NOT_AUTHORIZED"},
	ErrLimitExceeded:       {Status: 429, Code: "LIMIT_EXCEEDED"},
	ErrPasswordResetRequired: {Status: 403, Code: "PASSWORD_RESET_REQUIRED"},
	ErrInvalidParameter:    {Status: 400, Code: "INVALID_PARAMETER"},
}

// LookupError checks if the given error matches any known Cognito sentinel error
// and returns the corresponding ErrorInfo. Returns false if no match.
func LookupError(err error) (ErrorInfo, bool) {
	for sentinel, info := range errorMap {
		if errors.Is(err, sentinel) {
			return info, true
		}
	}
	return ErrorInfo{}, false
}
