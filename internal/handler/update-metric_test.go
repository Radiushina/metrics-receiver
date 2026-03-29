package handler

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

type mockStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockStorage) SetGauge(name string, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = v
}

func (m *mockStorage) AddCounter(name string, d int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += d
}

func (m *mockStorage) gauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges[name]
}

func (m *mockStorage) counter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func newTestMux(store repository.Storage) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("POST /update/{mtype}/{name}/{value}", NewUpdateMetricsHandler(store))
	return mux
}

func TestHandler_PostGauge_OK(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapAlloc/42.5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	if store.gauge("HeapAlloc") != 42.5 {
		t.Fatalf("stored gauge: %v", store.gauge("HeapAlloc"))
	}
}

func TestHandler_PostCounter_OK(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/3", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if store.counter("PollCount") != 3 {
		t.Fatalf("stored counter: %v", store.counter("PollCount"))
	}
}

func TestServeUpdateMetrics_EmptyName_NotFound(t *testing.T) {
	store := newMockStorage()
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/_/1", nil)
	req.SetPathValue("mtype", "gauge")
	req.SetPathValue("name", "")
	req.SetPathValue("value", "1")
	rec := httptest.NewRecorder()
	serveUpdateMetrics(rec, req, store)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_InvalidGaugeValue_BadRequest(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/x/not-a-float", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_InvalidCounterValue_BadRequest(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/x/1.5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_InvalidType_BadRequest(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodPost, "/update/unknown/m/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_GetMethod_MethodNotAllowed(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/x/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for GET, got %d", rec.Code)
	}
}
