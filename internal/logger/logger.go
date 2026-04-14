package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Log будет доступен всему коду как синглтон.
// Никакой код навыка, кроме функции Initialize, не должен модифицировать эту переменную.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log *zap.Logger = zap.NewNop()

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	if lw.wroteHeader {
		return
	}
	lw.status = code
	lw.wroteHeader = true
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lw.wroteHeader {
		lw.WriteHeader(http.StatusOK)
	}
	n, err := lw.ResponseWriter.Write(b)
	lw.bytes += n
	return n, err
}

// LoggingMiddleware пишет в лог сведения о запросе (URI, метод, длительность)
// и ответе (код статуса, размер тела).
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lw, r)
		uri := r.URL.RequestURI()
		Log.Info("HTTP",
			zap.String("uri", uri),
			zap.String("method", r.Method),
			zap.Duration("duration", time.Since(start)),
			zap.Int("status", lw.status),
			zap.Int("response_size", lw.bytes),
		)
	})
}
