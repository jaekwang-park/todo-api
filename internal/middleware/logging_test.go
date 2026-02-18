package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/middleware"
)

func TestLogging_LogsRequestInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := middleware.Logging(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	logOutput := buf.String()
	for _, want := range []string{"GET", "/health", "200"} {
		if !strings.Contains(logOutput, want) {
			t.Errorf("expected log to contain %q, got: %s", want, logOutput)
		}
	}
}

func TestLogging_DefaultStatusWhenNoExplicitWriteHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write body without calling WriteHeader — Go implicitly sends 200
		_, _ = w.Write([]byte("ok"))
	})

	h := middleware.Logging(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/implicit", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "200") {
		t.Errorf("expected log to contain status 200, got: %s", logOutput)
	}
}

func TestLogging_NonOKStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	h := middleware.Logging(logger)(inner)
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "404") {
		t.Errorf("expected log to contain status 404, got: %s", logOutput)
	}
}
