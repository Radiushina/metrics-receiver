package middleware

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Recover intercepts panics during request handling, logs them,
// and replies with HTTP 500 instead of crashing the process.
func Recover(log *zap.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic recovered",
						zap.Any("panic", v),
						zap.String("method", r.Method),
						zap.String("uri", r.URL.RequestURI()),
					)
					http.Error(w, fmt.Sprintf("internal server error: %v", v), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
