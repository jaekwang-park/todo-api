package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type JWKSClient struct {
	url        string
	mu         sync.RWMutex
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
	httpClient *http.Client
}

func NewJWKSClient(url string) *JWKSClient {
	return &JWKSClient{
		url:  url,
		keys: make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *JWKSClient) GetKey(kid string) (*rsa.PublicKey, error) {
	// Try cache first
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()
	if ok {
		return key, nil
	}

	// Rate-limit JWKS refreshes to prevent DoS via fabricated kid values
	c.mu.RLock()
	canRefresh := time.Since(c.lastFetch) > 5*time.Minute
	c.mu.RUnlock()

	if !canRefresh {
		return nil, fmt.Errorf("key with kid %q not found in JWKS", kid)
	}

	// Cache miss — fetch and retry
	if err := c.refresh(); err != nil {
		return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
	}

	c.mu.RLock()
	key, ok = c.keys[kid]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("key with kid %q not found in JWKS", kid)
	}

	return key, nil
}

func (c *JWKSClient) refresh() error {
	resp, err := c.httpClient.Get(c.url)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pubKey, err := parseRSAPublicKey(k)
		if err != nil {
			continue
		}
		newKeys[k.Kid] = pubKey
	}

	c.mu.Lock()
	c.keys = newKeys
	c.lastFetch = time.Now()
	c.mu.Unlock()

	return nil
}

func parseRSAPublicKey(k jwksKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
