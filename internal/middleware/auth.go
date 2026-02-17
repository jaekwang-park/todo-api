package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type AuthConfig struct {
	DevMode     bool
	JWKSClient  *JWKSClient
	Issuer      string
	AppClientID string
}

type Auth struct {
	cfg AuthConfig
}

func NewAuth(cfg AuthConfig) *Auth {
	return &Auth{cfg: cfg}
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and auth endpoints
		cleanPath := path.Clean(r.URL.Path)
		if cleanPath == "/health" || strings.HasPrefix(cleanPath, "/api/v1/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		if a.cfg.DevMode {
			a.handleDevMode(w, r, next)
			return
		}

		a.handleJWT(w, r, next)
	})
}

func (a *Auth) handleDevMode(w http.ResponseWriter, r *http.Request, next http.Handler) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "X-User-ID header required in dev mode")
		return
	}

	ctx := SetUserID(r.Context(), userID)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func (a *Auth) handleJWT(w http.ResponseWriter, r *http.Request, next http.Handler) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authorization header required")
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		return a.cfg.JWKSClient.GetKey(kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(a.cfg.Issuer),
		jwt.WithAudience(a.cfg.AppClientID),
	)

	if err != nil || !token.Valid {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
		return
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "sub claim not found")
		return
	}

	ctx := SetUserID(r.Context(), sub)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// CognitoJWKSURL returns the JWKS URL for the given Cognito User Pool.
func CognitoJWKSURL(region, userPoolID string) string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID)
}

// CognitoIssuer returns the expected issuer for the given Cognito User Pool.
func CognitoIssuer(region, userPoolID string) string {
	return fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", region, userPoolID)
}

