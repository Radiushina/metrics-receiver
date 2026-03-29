package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/go-chi/chi/v5"
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

func (m *mockStorage) GetGauge(name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *mockStorage) GetCounter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.counters[name]
	return v, ok
}

func (m *mockStorage) Gauges() map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]float64, len(m.gauges))
	for k, v := range m.gauges {
		out[k] = v
	}
	return out
}

func newTestMux(store repository.Storage) http.Handler {
	r := chi.NewRouter()
	r.Get("/", NewMetricHandler(store))
	r.Post("/update/{mtype}/{name}/{value}", NewUpdateMetricsHandler(store))
	r.Get("/value/{mtype}/{name}", NewValueHandler(store))
	return r
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
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("mtype", "gauge")
	rctx.URLParams.Add("name", "")
	rctx.URLParams.Add("value", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
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
		t.Fatalf("expected 405 for GET on POST-only route, got %d", rec.Code)
	}
}

func TestHandler_GetValue_Gauge_OK(t *testing.T) {
	store := newMockStorage()
	store.SetGauge("HeapAlloc", 42.5)
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/HeapAlloc", nil)
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
	store := newMockStorage()
	store.AddCounter("PollCount", 7)
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
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
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/NoSuch", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_Root_HTML_Empty(t *testing.T) {
	store := newMockStorage()
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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

func TestHandler_Root_HTML_ListsMetrics(t *testing.T) {
	store := newMockStorage()
	store.SetGauge("HeapAlloc", 1.25)
	store.AddCounter("PollCount", 4)
	mux := newTestMux(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "HeapAlloc") || !strings.Contains(body, "PollCount") {
		t.Fatalf("missing names in body %q", body)
	}
	wantG := strconv.FormatFloat(1.25, 'g', -1, 64)
	if !strings.Contains(body, wantG) || !strings.Contains(body, "PollCount: 4") {
		t.Fatalf("missing values in body %q", body)
	}
}
