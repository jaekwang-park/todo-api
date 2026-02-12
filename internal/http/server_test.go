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

func TestServer_StartAndShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	port := freePort(t)
	srv := todohttp.NewServer(port, logger, newTestTodoSvc())

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
