// Package middleware содержит HTTP-middleware: recovery, gzip-сжатие и распаковку.
package middleware

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// Recover перехватывает panic при обработке запроса, логирует и отвечает HTTP 500.
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
