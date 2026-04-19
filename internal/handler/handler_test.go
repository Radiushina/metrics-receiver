package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-chi/chi/v5"
)

type mockService struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func newMockService() *mockService {
	return &mockService{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockService) SetGauge(name string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = v
}

func (m *mockService) AddCounter(name string, d int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += d
}

func (m *mockService) gauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges[name]
}

func (m *mockService) counter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func (m *mockService) GetGauge(name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *mockService) GetCounter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.counters[name]
	return v, ok
}

func (m *mockService) Gauges() map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]float64, len(m.gauges))
	for k, v := range m.gauges {
		out[k] = v
	}
	return out
}

func (m *mockService) Counters() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		out[k] = v
	}
	return out
}

func newTestMux(svc handler.ServiceProvider) http.Handler {
	h := handler.NewHandler(svc)
	r := chi.NewRouter()
	r.Get("/", h.GetMetrics())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	return r
}

func TestHandler_PostGauge_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/gauge/HeapAlloc/42.5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	if svc.gauge("HeapAlloc") != 42.5 {
		t.Fatalf("stored gauge: %v", svc.gauge("HeapAlloc"))
	}
}

func TestHandler_PostCounter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/counter/PollCount/3", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if svc.counter(models.PollCount) != 3 {
		t.Fatalf("stored counter: %v", svc.counter("PollCount"))
	}
}

func TestHandler_InvalidGaugeValue_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/gauge/x/not-a-float", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_InvalidCounterValue_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/counter/x/1.5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_InvalidType_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/unknown/m/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetMethod_MethodNotAllowed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/update/gauge/x/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET on POST-only route, got %d", rec.Code)
	}
}

func TestHandler_GetValue_Gauge_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	svc.SetGauge("HeapAlloc", 42.5)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/value/gauge/HeapAlloc", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	want := strconv.FormatFloat(42.5, 'g', -1, 64)
	if got := rec.Body.String(); got != want {
		t.Fatalf("body %q, want %q", got, want)
	}
}

func TestHandler_GetValue_Counter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	svc.AddCounter(models.PollCount, 7)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/value/counter/PollCount", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "7" {
		t.Fatalf("body %q, want 7", got)
	}
}

func TestHandler_GetValue_Unknown_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/value/gauge/NoSuch", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_GetAll_HTML_Empty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No metrics yet.") {
		t.Fatalf("body %q", body)
	}
}

func TestHandler_GetAll_HTML_ListsMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	svc.SetGauge("HeapAlloc", 1.25)
	svc.AddCounter(models.PollCount, 4)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "HeapAlloc") || !strings.Contains(body, models.PollCount) {
		t.Fatalf("missing names in body %q", body)
	}
	wantG := strconv.FormatFloat(1.25, 'g', -1, 64)
	if !strings.Contains(body, wantG) || !strings.Contains(body, "PollCount: 4") {
		t.Fatalf("missing values in body %q", body)
	}
}

func TestHandler_GetAll_HTML_AllCountersListed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	svc.AddCounter(models.PollCount, 1)
	svc.AddCounter("OtherCounter", 2)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "PollCount: 1") || !strings.Contains(body, "OtherCounter: 2") {
		t.Fatalf("expected both counters in body %q", body)
	}
}
