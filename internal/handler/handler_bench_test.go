package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// benchHandler собирает handler на in-memory репозитории без audit и file storage.
func benchHandler() *handler.Handler {
	repo := repository.NewMemoryRepo()
	svc := service.NewService(repo)
	return handler.NewHandler(svc, nil, zap.NewNop(), nil, "", nil)
}

// benchRouter регистрирует маршруты, участвующие в бенчмарках.
func benchRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.GetMetrics())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Post("/update", h.UpdateFromBody())
	r.Post("/updates/", h.UpdateMetrics())
	return r
}

// benchAgentBatchJSON — JSON-тело POST /updates/, как отправляет агент.
func benchAgentBatchJSON(b *testing.B) []byte {
	b.Helper()
	metrics := make([]models.Metrics, 0, len(models.GaugeNames)+len(models.CounterNames))
	for _, name := range models.GaugeNames {
		v := 1.0
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}
	for _, name := range models.CounterNames {
		d := int64(1)
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &d,
		})
	}
	body, err := json.Marshal(metrics)
	if err != nil {
		b.Fatal(err)
	}
	return body
}

// BenchmarkHandlerUpdateFromBody измеряет POST /update с JSON-телом (одна метрика).
func BenchmarkHandlerUpdateFromBody(b *testing.B) {
	mux := benchRouter(benchHandler())
	body := []byte(`{"id":"HeapAlloc","type":"gauge","value":42.5}`)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}

// BenchmarkHandlerUpdateFromPath измеряет POST /update/{mtype}/{metric}/{value} (метрика в URL).
func BenchmarkHandlerUpdateFromPath(b *testing.B) {
	mux := benchRouter(benchHandler())

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapAlloc/42.5", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}

// BenchmarkHandlerUpdateMetricsBatch измеряет POST /updates/ — пакетная отправка с агента.
func BenchmarkHandlerUpdateMetricsBatch(b *testing.B) {
	mux := benchRouter(benchHandler())
	body := benchAgentBatchJSON(b)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}

// BenchmarkHandlerGetMetricsIndex измеряет GET / — HTML-страница со списком всех метрик.
func BenchmarkHandlerGetMetricsIndex(b *testing.B) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()
	for _, name := range models.GaugeNames {
		_ = repo.SetGauge(ctx, name, 1)
	}
	_ = repo.AddCounter(ctx, models.PollCount, 1)
	mux := benchRouter(handler.NewHandler(service.NewService(repo), nil, zap.NewNop(), nil, "", nil))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}
