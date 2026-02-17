package middleware_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/middleware"
)

// helper: generate RSA key pair and return JWKS JSON + the private key
func generateTestJWKS(t *testing.T, kid string) ([]byte, *rsa.PrivateKey) {
	t.Helper()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

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

	return data, privKey
}

func TestJWKSClient_FetchKey(t *testing.T) {
	jwksData, privKey := generateTestJWKS(t, "test-kid-1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksData)
	}))
	defer server.Close()

	client := middleware.NewJWKSClient(server.URL)

	pubKey, err := client.GetKey("test-kid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pubKey.N.Cmp(privKey.N) != 0 {
		t.Error("public key N does not match private key N")
	}
}

func TestJWKSClient_KeyNotFound(t *testing.T) {
	jwksData, _ := generateTestJWKS(t, "test-kid-1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksData)
	}))
	defer server.Close()

	client := middleware.NewJWKSClient(server.URL)

	_, err := client.GetKey("nonexistent-kid")
	if err == nil {
		t.Fatal("expected error for missing kid, got nil")
	}
}

func TestJWKSClient_CachesKeys(t *testing.T) {
	jwksData, _ := generateTestJWKS(t, "cached-kid")

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksData)
	}))
	defer server.Close()

	client := middleware.NewJWKSClient(server.URL)

	// First call fetches
	_, err := client.GetKey("cached-kid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should use cache
	_, err = client.GetKey("cached-kid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 fetch call, got %d", callCount)
	}
}

func TestJWKSClient_RateLimitRefreshOnMissingKid(t *testing.T) {
	jwksData, _ := generateTestJWKS(t, "kid-v1")

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksData)
	}))
	defer server.Close()

	client := middleware.NewJWKSClient(server.URL)

	// Prime the cache
	_, _ = client.GetKey("kid-v1")

	// Request a different kid → should NOT trigger refresh (rate limited)
	_, err := client.GetKey("kid-v2")
	if err == nil {
		t.Fatal("expected error for missing kid")
	}

	// Should have fetched only once: initial. Second is rate-limited.
	if callCount != 1 {
		t.Errorf("expected 1 fetch call (rate limited), got %d", callCount)
	}
}

func TestJWKSClient_FetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := middleware.NewJWKSClient(server.URL)

	_, err := client.GetKey("any-kid")
	if err == nil {
		t.Fatal("expected error on server error, got nil")
	}
}
