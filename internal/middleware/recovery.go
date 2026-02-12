package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

type recoveryWriter struct {
	http.ResponseWriter
	headerWritten bool
}

func (rw *recoveryWriter) WriteHeader(code int) {
	rw.headerWritten = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *recoveryWriter) Write(b []byte) (int, error) {
	rw.headerWritten = true
	return rw.ResponseWriter.Write(b)
}

func (rw *recoveryWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &recoveryWriter{ResponseWriter: w}

			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)

					if rw.headerWritten {
						return
					}

					rw.Header().Set("Content-Type", "application/json")
					rw.WriteHeader(http.StatusInternalServerError)
					if encErr := json.NewEncoder(rw).Encode(map[string]any{
						"error": map[string]string{
							"code":    "INTERNAL_ERROR",
							"message": "internal server error",
						},
					}); encErr != nil {
						logger.Error("failed to write recovery response", "error", encErr)
					}
				}
			}()

			next.ServeHTTP(rw, r)
		})
	}
}
