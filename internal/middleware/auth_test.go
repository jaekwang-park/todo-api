package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jaekwang-park/todo-api/internal/middleware"
)

func signedToken(t *testing.T, privKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func jwksServer(t *testing.T, kid string, privKey *rsa.PrivateKey) *httptest.Server {
	t.Helper()
	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(privKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privKey.E)).Bytes()),
			},
		},
	}
	data, err := json.Marshal(jwks)
	if err != nil {
		t.Fatalf("failed to marshal JWKS: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func generateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return privKey
}

type mockUserResolver struct {
	userID     string
	err        error
	calledWith string
}

func (m *mockUserResolver) ResolveUserID(_ context.Context, cognitoSub string) (string, error) {
	m.calledWith = cognitoSub
	return m.userID, m.err
}

func mustNewAuth(t *testing.T, cfg middleware.AuthConfig) *middleware.Auth {
	t.Helper()
	auth, err := middleware.NewAuth(cfg)
	if err != nil {
		t.Fatalf("NewAuth failed: %v", err)
	}
	return auth
}

// jwtAuthConfig returns a valid JWT-mode AuthConfig with the given resolver.
func jwtAuthConfig(jwksURL string, resolver middleware.UserResolver) middleware.AuthConfig {
	return middleware.AuthConfig{
		DevMode:      false,
		JWKSClient:   middleware.NewJWKSClient(jwksURL),
		Issuer:       "https://cognito-idp.ap-northeast-2.amazonaws.com/pool-1",
		AppClientID:  "client-1",
		UserResolver: resolver,
	}
}

func TestAuth_DevMode(t *testing.T) {
	cfg := middleware.AuthConfig{DevMode: true}
	auth := mustNewAuth(t, cfg)

	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = middleware.GetUserID(r)
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		userIDHdr  string
		wantStatus int
		wantUserID string
	}{
		{"with X-User-ID", "dev-user-1", http.StatusOK, "dev-user-1"},
		{"without X-User-ID", "", http.StatusUnauthorized, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedUserID = ""
			req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
			if tt.userIDHdr != "" {
				req.Header.Set("X-User-ID", tt.userIDHdr)
			}
			w := httptest.NewRecorder()

			auth.Middleware(inner).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK && capturedUserID != tt.wantUserID {
				t.Errorf("expected userID=%q, got %q", tt.wantUserID, capturedUserID)
			}
		})
	}
}

func TestAuth_DevMode_SkipsHealthCheck(t *testing.T) {
	cfg := middleware.AuthConfig{DevMode: true}
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}
}

func TestAuth_SkipsAuthEndpoints(t *testing.T) {
	resolver := &mockUserResolver{userID: "unused"}

	tests := []struct {
		name    string
		devMode bool
	}{
		{"dev mode", true},
		{"jwt mode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg middleware.AuthConfig
			if tt.devMode {
				cfg = middleware.AuthConfig{DevMode: true}
			} else {
				cfg = jwtAuthConfig("http://unused", resolver)
			}
			auth := mustNewAuth(t, cfg)

			var called bool
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			paths := []string{
				"/api/v1/auth/signup",
				"/api/v1/auth/login",
				"/api/v1/auth/confirm-signup",
				"/api/v1/auth/refresh",
				"/api/v1/auth/forgot-password",
			}

			for _, path := range paths {
				called = false
				req := httptest.NewRequest(http.MethodPost, path, nil)
				w := httptest.NewRecorder()

				auth.Middleware(inner).ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("%s: expected 200, got %d", path, w.Code)
				}
				if !called {
					t.Errorf("%s: inner handler was not called", path)
				}
			}
		})
	}
}

func TestAuth_JWT_Valid(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{userID: "db-user-uuid-abc"}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = middleware.GetUserID(r)
		w.WriteHeader(http.StatusOK)
	})

	token := signedToken(t, privKey, kid, jwt.MapClaims{
		"sub":       "cognito-sub-123",
		"iss":       "https://cognito-idp.ap-northeast-2.amazonaws.com/pool-1",
		"aud":       "client-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"token_use": "id",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedUserID != "db-user-uuid-abc" {
		t.Errorf("expected userID=db-user-uuid-abc, got %q", capturedUserID)
	}
	if resolver.calledWith != "cognito-sub-123" {
		t.Errorf("expected resolver called with cognito-sub-123, got %q", resolver.calledWith)
	}
}

func TestAuth_JWT_UserNotFound(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{err: middleware.ErrUserNotFound}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	token := signedToken(t, privKey, kid, jwt.MapClaims{
		"sub":       "cognito-sub-unknown",
		"iss":       "https://cognito-idp.ap-northeast-2.amazonaws.com/pool-1",
		"aud":       "client-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"token_use": "id",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_JWT_ResolverInternalError(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{err: fmt.Errorf("connection refused")}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	token := signedToken(t, privKey, kid, jwt.MapClaims{
		"sub":       "cognito-sub-123",
		"iss":       "https://cognito-idp.ap-northeast-2.amazonaws.com/pool-1",
		"aud":       "client-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"token_use": "id",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAuth_JWT_MissingHeader(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{userID: "unused"}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_JWT_ExpiredToken(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{userID: "unused"}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	token := signedToken(t, privKey, kid, jwt.MapClaims{
		"sub":       "cognito-sub-123",
		"iss":       "https://cognito-idp.ap-northeast-2.amazonaws.com/pool-1",
		"aud":       "client-1",
		"exp":       time.Now().Add(-time.Hour).Unix(), // expired
		"token_use": "id",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_JWT_WrongIssuer(t *testing.T) {
	privKey := generateKey(t)
	kid := "jwt-test-kid"
	srv := jwksServer(t, kid, privKey)

	resolver := &mockUserResolver{userID: "unused"}
	cfg := jwtAuthConfig(srv.URL, resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	token := signedToken(t, privKey, kid, jwt.MapClaims{
		"sub": "cognito-sub-123",
		"iss": "https://wrong-issuer.example.com",
		"aud": "client-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_JWT_InvalidBearerFormat(t *testing.T) {
	resolver := &mockUserResolver{userID: "unused"}
	cfg := jwtAuthConfig("http://unused", resolver)
	auth := mustNewAuth(t, cfg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	req.Header.Set("Authorization", "NotBearer token")
	w := httptest.NewRecorder()

	auth.Middleware(inner).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestNewAuth_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     middleware.AuthConfig
		wantErr bool
	}{
		{
			name:    "dev mode requires no resolver",
			cfg:     middleware.AuthConfig{DevMode: true},
			wantErr: false,
		},
		{
			name: "jwt mode without resolver fails",
			cfg: middleware.AuthConfig{
				DevMode:    false,
				JWKSClient: middleware.NewJWKSClient("http://unused"),
			},
			wantErr: true,
		},
		{
			name: "jwt mode without JWKS client fails",
			cfg: middleware.AuthConfig{
				DevMode:      false,
				UserResolver: &mockUserResolver{userID: "unused"},
			},
			wantErr: true,
		},
		{
			name: "jwt mode with all deps succeeds",
			cfg: middleware.AuthConfig{
				DevMode:      false,
				JWKSClient:   middleware.NewJWKSClient("http://unused"),
				UserResolver: &mockUserResolver{userID: "unused"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := middleware.NewAuth(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuth() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
