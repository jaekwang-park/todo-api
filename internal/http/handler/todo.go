package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jaekwang-park/todo-api/internal/middleware"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/service"
)

type TodoHandler struct {
	svc *service.TodoService
}

func NewTodoHandler(svc *service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// ServeHTTP routes /api/v1/todos and /api/v1/todos/{id}
func (h *TodoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract todo ID from path: /api/v1/todos/{id}/...
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/todos")
	path = strings.TrimPrefix(path, "/")

	parts := strings.SplitN(path, "/", 2)
	todoID := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = parts[1]
	}

	// /api/v1/todos/{id}/status
	if todoID != "" && subPath == "status" {
		h.handleUpdateStatus(w, r, todoID)
		return
	}

	// /api/v1/todos/{id}
	if todoID != "" {
		switch r.Method {
		case http.MethodGet:
			h.handleGetByID(w, r, todoID)
		case http.MethodPut:
			h.handleUpdate(w, r, todoID)
		case http.MethodDelete:
			h.handleDelete(w, r, todoID)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// /api/v1/todos
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

type createTodoRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueAt       *string `json:"due_at,omitempty"`
}

func (h *TodoHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req createTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	input := service.CreateTodoInput{
		Title:       req.Title,
		Description: req.Description,
		DueAt:       req.DueAt,
	}

	todo, err := h.svc.Create(r.Context(), userID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, todo)
}

func (h *TodoHandler) handleGetByID(w http.ResponseWriter, r *http.Request, todoID string) {
	userID := getUserID(r)

	todo, err := h.svc.GetByID(r.Context(), userID, todoID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, todo)
}

type updateTodoRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DueAt       *string `json:"due_at,omitempty"`
}

func (h *TodoHandler) handleUpdate(w http.ResponseWriter, r *http.Request, todoID string) {
	userID := getUserID(r)

	var req updateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	input := service.UpdateTodoInput{
		Title:       req.Title,
		Description: req.Description,
		DueAt:       req.DueAt,
	}

	todo, err := h.svc.Update(r.Context(), userID, todoID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) handleDelete(w http.ResponseWriter, r *http.Request, todoID string) {
	userID := getUserID(r)

	if err := h.svc.Delete(r.Context(), userID, todoID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *TodoHandler) handleUpdateStatus(w http.ResponseWriter, r *http.Request, todoID string) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	userID := getUserID(r)

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	todo, err := h.svc.UpdateStatus(r.Context(), userID, todoID, model.TodoStatus(req.Status))
	if err != nil {
		handleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) handleList(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	params := model.TodoListParams{
		UserID: userID,
		Cursor: r.URL.Query().Get("cursor"),
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := model.TodoStatus(statusStr)
		if !status.IsValid() {
			WriteError(w, http.StatusBadRequest, "INVALID_STATUS", "status must be 'pending' or 'completed'")
			return
		}
		params.Status = &status
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	params.Limit = limit

	result, err := h.svc.List(r.Context(), params)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func getUserID(r *http.Request) string {
	return middleware.GetUserID(r)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, service.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, service.ErrForbidden):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
