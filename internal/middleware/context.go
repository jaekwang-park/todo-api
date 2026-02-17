package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "user_id"

func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(r *http.Request) string {
	v, _ := r.Context().Value(userIDKey).(string)
	return v
}
