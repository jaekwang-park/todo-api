package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	todohttp "github.com/jaekwang-park/todo-api/internal/http"
	"github.com/jaekwang-park/todo-api/internal/middleware"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

func newDevAuth() *middleware.Auth {
	return middleware.NewAuth(middleware.AuthConfig{DevMode: true})
}

func TestServer_StartAndShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	port := freePort(t)
	srv := todohttp.NewServer(port, logger, newTestTodoSvc(), newTestAuthSvc(), newDevAuth())

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	// Wait for server to be ready
	addr := fmt.Sprintf("http://localhost:%s/health", port)
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, _ = http.Get(addr)
		if resp != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if resp == nil {
		t.Fatal("server did not start in time")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestServer_AuthMiddlewareApplied(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	port := freePort(t)
	srv := todohttp.NewServer(port, logger, newTestTodoSvc(), newTestAuthSvc(), newDevAuth())

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	// Wait for server to be ready
	addr := fmt.Sprintf("http://localhost:%s/health", port)
	for i := 0; i < 50; i++ {
		if resp, _ := http.Get(addr); resp != nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Request to /api/v1/todos without X-User-ID → should get 401
	todosAddr := fmt.Sprintf("http://localhost:%s/api/v1/todos", port)
	resp, err := http.Get(todosAddr)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}

	// Request with X-User-ID → should succeed
	req, _ := http.NewRequest(http.MethodGet, todosAddr, nil)
	req.Header.Set("X-User-ID", "test-user")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with auth, got %d", resp2.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
