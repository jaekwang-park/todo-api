package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/http/handler"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	handler.WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	handler.WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "name is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var result handler.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected code=INVALID_INPUT, got %s", result.Error.Code)
	}
	if result.Error.Message != "name is required" {
		t.Errorf("expected message='name is required', got %s", result.Error.Message)
	}
}
