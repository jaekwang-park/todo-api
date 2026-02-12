package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/middleware"
)

func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	return logger, &buf
}

func TestRecovery_NoPanic(t *testing.T) {
	logger, _ := newTestLogger()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := middleware.Recovery(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRecovery_WithPanic(t *testing.T) {
	logger, _ := newTestLogger()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	h := middleware.Recovery(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected code=INTERNAL_ERROR, got %v", errObj["code"])
	}
	if errObj["message"] != "internal server error" {
		t.Errorf("expected message='internal server error', got %v", errObj["message"])
	}
}

func TestRecovery_PanicAfterHeaderWritten(t *testing.T) {
	logger, logBuf := newTestLogger()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"partial"}`))
		panic("panic after write")
	})

	h := middleware.Recovery(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	// Status should remain 200 since header was already written
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 (already written), got %d", w.Code)
	}

	// Recovery should have logged the panic
	if !bytes.Contains(logBuf.Bytes(), []byte("panic recovered")) {
		t.Error("expected panic to be logged")
	}
}
