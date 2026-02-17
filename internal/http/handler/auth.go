package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jaekwang-park/todo-api/internal/cognito"
	"github.com/jaekwang-park/todo-api/internal/service"
)

const maxAuthBodySize = 1 << 20 // 1 MB

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// ServeHTTP routes /api/v1/auth/* requests.
func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/")
	path = strings.TrimRight(path, "/")

	switch path {
	case "signup":
		h.requirePost(w, r, h.handleSignUp)
	case "confirm-signup":
		h.requirePost(w, r, h.handleConfirmSignUp)
	case "resend-code":
		h.requirePost(w, r, h.handleResendCode)
	case "login":
		h.requirePost(w, r, h.handleLogin)
	case "refresh":
		h.requirePost(w, r, h.handleRefresh)
	case "forgot-password":
		h.requirePost(w, r, h.handleForgotPassword)
	case "confirm-forgot-password":
		h.requirePost(w, r, h.handleConfirmForgotPassword)
	case "change-password":
		h.requirePost(w, r, h.handleChangePassword)
	case "logout":
		h.requirePost(w, r, h.handleLogout)
	default:
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
	}
}

func (h *AuthHandler) requirePost(w http.ResponseWriter, r *http.Request, handler func(http.ResponseWriter, *http.Request)) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodySize)
	handler(w, r)
}

// --- DTOs ---

type signUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type confirmSignUpRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type resendCodeRequest struct {
	Email string `json:"email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type confirmForgotPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

type changePasswordRequest struct {
	AccessToken      string `json:"access_token"`
	PreviousPassword string `json:"previous_password"`
	NewPassword      string `json:"new_password"`
}

type logoutRequest struct {
	AccessToken string `json:"access_token"`
}

// --- Handlers ---

func (h *AuthHandler) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req signUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	out, err := h.svc.SignUp(r.Context(), service.SignUpInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, out)
}

func (h *AuthHandler) handleConfirmSignUp(w http.ResponseWriter, r *http.Request) {
	var req confirmSignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.ConfirmSignUp(r.Context(), service.ConfirmSignUpInput{
		Email: req.Email,
		Code:  req.Code,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "email confirmed"})
}

func (h *AuthHandler) handleResendCode(w http.ResponseWriter, r *http.Request) {
	var req resendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.ResendCode(r.Context(), service.ResendCodeInput{
		Email: req.Email,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "confirmation code resent"})
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	out, err := h.svc.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *AuthHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	out, err := h.svc.Refresh(r.Context(), service.RefreshInput{
		Email:        req.Email,
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *AuthHandler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.ForgotPassword(r.Context(), service.ForgotPasswordInput{
		Email: req.Email,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "password reset code sent"})
}

func (h *AuthHandler) handleConfirmForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req confirmForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.ConfirmForgotPassword(r.Context(), service.ConfirmForgotPasswordInput{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "password reset confirmed"})
}

func (h *AuthHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), service.ChangePasswordInput{
		AccessToken:      req.AccessToken,
		PreviousPassword: req.PreviousPassword,
		NewPassword:      req.NewPassword,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "password changed"})
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := h.svc.Logout(r.Context(), service.LogoutInput{
		AccessToken: req.AccessToken,
	}); err != nil {
		handleAuthError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "signed out"})
}

// handleAuthError maps cognito sentinel errors and service errors to HTTP responses.
// Uses fixed messages to avoid leaking internal error details to clients.
// Logs actual error details server-side for debugging.
func handleAuthError(w http.ResponseWriter, err error) {
	// Check cognito sentinel errors first
	if info, ok := cognito.LookupError(err); ok {
		slog.Error("auth error", "code", info.Code, "detail", err.Error())
		WriteError(w, info.Status, info.Code, cognitoErrorMessage(info.Code))
		return
	}

	// Check service-level errors
	if errors.Is(err, service.ErrInvalidInput) {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	slog.Error("auth internal error", "error", err.Error())
	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

// cognitoErrorMessage returns a safe, user-facing message for each cognito error code.
func cognitoErrorMessage(code string) string {
	messages := map[string]string{
		"USER_ALREADY_EXISTS":   "a user with this email already exists",
		"USER_NOT_FOUND":        "user not found",
		"USER_NOT_CONFIRMED":    "email address not confirmed",
		"INVALID_PASSWORD":      "password does not meet requirements",
		"INVALID_CODE":          "invalid verification code",
		"CODE_EXPIRED":          "verification code has expired",
		"TOO_MANY_REQUESTS":     "too many requests, please try again later",
		"NOT_AUTHORIZED":        "incorrect email or password",
		"LIMIT_EXCEEDED":        "attempt limit exceeded, please try again later",
		"PASSWORD_RESET_REQUIRED": "password reset is required",
		"INVALID_PARAMETER":     "invalid request parameter",
	}
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "an error occurred"
}
