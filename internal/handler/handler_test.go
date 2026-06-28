package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"go.uber.org/zap"
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

func (m *mockService) SetGauge(_ context.Context, name string, v float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = v
	return nil
}

func (m *mockService) AddCounter(_ context.Context, name string, d int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += d
	return nil
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

func (m *mockService) GetGauge(_ context.Context, name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *mockService) GetCounter(_ context.Context, name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.counters[name]
	return v, ok
}

func (m *mockService) Gauges(_ context.Context) map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]float64, len(m.gauges))
	for k, v := range m.gauges {
		out[k] = v
	}
	return out
}

func (m *mockService) Counters(_ context.Context) map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		out[k] = v
	}
	return out
}

func (m *mockService) UpdateMetricsBatch(_ context.Context, metrics []models.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				continue
			}
			m.gauges[metric.ID] = *metric.Value
		case models.Counter:
			if metric.Delta == nil {
				continue
			}
			m.counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}

type mockDBChecker struct {
	err error
}

func (m *mockDBChecker) Ping(ctx context.Context) error {
	return m.err
}

func newPingMux(db handler.DBChecker) http.Handler {
	h := handler.NewHandler(newMockService(), nil, zap.NewNop(), db, "", nil)
	r := chi.NewRouter()
	r.Get("/ping", h.PingDB())
	return r
}

func newTestMux(svc handler.ServiceProvider) http.Handler {
	return newTestMuxWithKey(svc, "")
}

func newTestMuxWithKey(svc handler.ServiceProvider, secretKey string) http.Handler {
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil, secretKey, nil)
	r := chi.NewRouter()
	r.Get("/", h.GetMetrics())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Post("/update", h.UpdateFromBody())
	r.Post("/update/", h.UpdateFromBody())
	r.Post("/updates/", h.UpdateMetrics())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	r.Post("/value", h.GetMetricValue())
	r.Post("/value/", h.GetMetricValue())
	return r
}

type errService struct {
	handler.ServiceProvider
	err error
}

func (e errService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	return e.err
}

func TestHandler_PostGauge_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge","value":42.5}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	var got models.Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if got.ID != "HeapAlloc" || got.MType != models.Gauge || got.Value == nil || *got.Value != 42.5 {
		t.Fatalf("response body: %+v", got)
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

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"PollCount","type":"counter","delta":3}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var got models.Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if got.ID != models.PollCount || got.MType != models.Counter || got.Delta == nil || *got.Delta != 3 {
		t.Fatalf("response body: %+v", got)
	}
	if svc.counter(models.PollCount) != 3 {
		t.Fatalf("stored counter: %v", svc.counter("PollCount"))
	}
}

func TestHandler_PostUpdatesBatch_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/updates/", strings.NewReader(
		`[
  {"id":"PollCount","type":"counter","delta":42},
  {"id":"RandomValue","type":"gauge","value":3.14159},
  {"id":"ActiveUsers","type":"gauge","value":1547}
]`,
	))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}

	var got []models.Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 metrics, got %d: %+v", len(got), got)
	}

	if svc.counter(models.PollCount) != 42 {
		t.Fatalf("stored counter: %v", svc.counter(models.PollCount))
	}
	if svc.gauge("RandomValue") != 3.14159 {
		t.Fatalf("stored gauge RandomValue: %v", svc.gauge("RandomValue"))
	}
	if svc.gauge("ActiveUsers") != 1547 {
		t.Fatalf("stored gauge ActiveUsers: %v", svc.gauge("ActiveUsers"))
	}
}

func TestHandler_PostUpdatesBatch_ServiceError_InternalServerError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := errService{ServiceProvider: newMockService(), err: errors.New("db is down")}
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/updates/", strings.NewReader(
		`[
  {"id":"PollCount","type":"counter","delta":42},
  {"id":"RandomValue","type":"gauge","value":3.14159}
]`,
	))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestHandler_PostUpdatesBatch_InvalidMetric_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	// gauge без value — это ошибка валидации запроса => 400
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/updates/", strings.NewReader(
		`[
  {"id":"RandomValue","type":"gauge"}
]`,
	))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %q", rec.Code, rec.Body.String())
	}
}

func pathRequestHashSHA256B64(key, path string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(path))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func bodyHashSHA256B64(key string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func gzipBody(t *testing.T, plain []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(plain); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestHandler_UpdateFromBody_WithKey_hmacPlainJSON_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "secret-plain"
	mux := newTestMuxWithKey(newMockService(), secret)
	raw := []byte(`{"id":"HeapAlloc","type":"gauge","value":2}`)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", bodyHashSHA256B64(secret, raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestHandler_UpdateFromBody_WithKey_hmacOverGzipBytes_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "secret-gzip"
	mux := newTestMuxWithKey(newMockService(), secret)
	raw := []byte(`{"id":"HeapAlloc","type":"gauge","value":2}`)
	gz := gzipBody(t, raw)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HashSHA256", bodyHashSHA256B64(secret, gz))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestHandler_UpdateFromPath_WithKey_NoRequestHash_Allowed_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "test-secret-key"
	mux := newTestMuxWithKey(newMockService(), secret)
	path := "/update/gauge/HeapAlloc/42.5"

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %q", rec.Code, rec.Body.String())
	}
	wantHdr := bodyHashSHA256B64(secret, nil)
	if got := rec.Header().Get("HashSHA256"); got != wantHdr {
		t.Fatalf("HashSHA256 response header: got %q want %q", got, wantHdr)
	}
}

func TestHandler_UpdateFromBody_WithKey_NoRequestHash_Allowed_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "no-req-hash"
	mux := newTestMuxWithKey(newMockService(), secret)
	raw := []byte(`{"id":"HeapAlloc","type":"gauge","value":2}`)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %q", rec.Code, rec.Body.String())
	}
	respBody := rec.Body.Bytes()
	wantHdr := bodyHashSHA256B64(secret, respBody)
	if got := rec.Header().Get("HashSHA256"); got != wantHdr {
		t.Fatalf("HashSHA256 response header: got %q want %q", got, wantHdr)
	}
}

func TestHandler_UpdateFromPath_WithKey_WrongHash_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "test-secret-key"
	mux := newTestMuxWithKey(newMockService(), secret)
	path := "/update/gauge/HeapAlloc/42.5"

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	req.Header.Set("HashSHA256", "AAAA")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %q", rec.Code, rec.Body.String())
	}
	body := rec.Body.Bytes()
	wantHdr := bodyHashSHA256B64(secret, body)
	if got := rec.Header().Get("HashSHA256"); got != wantHdr {
		t.Fatalf("HashSHA256 header: got %q want %q", got, wantHdr)
	}
}

func TestHandler_UpdateFromPath_WithKey_OK_ResponseHash(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const secret = "test-secret-key"
	mux := newTestMuxWithKey(newMockService(), secret)
	path := "/update/gauge/HeapAlloc/42.5"

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	req.Header.Set("HashSHA256", pathRequestHashSHA256B64(secret, path))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %q", rec.Code, rec.Body.String())
	}
	wantHdr := bodyHashSHA256B64(secret, nil)
	if got := rec.Header().Get("HashSHA256"); got != wantHdr {
		t.Fatalf("HashSHA256 header: got %q want %q", got, wantHdr)
	}
}

func TestHandler_InvalidGaugeValue_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"x","type":"gauge","value":"not-a-float"}`,
	))
	req.Header.Set("Content-Type", "application/json")
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

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"x","type":"counter","delta":1.5}`,
	))
	req.Header.Set("Content-Type", "application/json")
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

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"m","type":"unknown","value":1}`,
	))
	req.Header.Set("Content-Type", "application/json")
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

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/update", nil)
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
	_ = svc.SetGauge(ctx, "HeapAlloc", 42.5)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/value", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q", ct)
	}
	var out models.Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != "HeapAlloc" || out.MType != models.Gauge || out.Value == nil || out.Delta != nil {
		t.Fatalf("response %+v", out)
	}
	if *out.Value != 42.5 {
		t.Fatalf("value %v", *out.Value)
	}
}

func TestHandler_GetValue_Counter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	_ = svc.AddCounter(ctx, models.PollCount, 7)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/value", strings.NewReader(
		`{"id":"PollCount","type":"counter"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	var out models.Metrics
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != models.PollCount || out.MType != models.Counter || out.Value != nil || out.Delta == nil || *out.Delta != 7 {
		t.Fatalf("response %+v", out)
	}
}

func TestHandler_GetValue_Unknown_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/value", strings.NewReader(
		`{"id":"NoSuch","type":"gauge"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_UpdatePath_Gauge_OK(t *testing.T) {
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

func TestHandler_UpdatePath_Counter_OK(t *testing.T) {
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

func TestHandler_GetPath_Gauge_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	_ = svc.SetGauge(ctx, "HeapAlloc", 42.5)
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

func TestHandler_GetPath_Counter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	svc := newMockService()
	_ = svc.AddCounter(ctx, models.PollCount, 7)
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

func TestHandler_GetPath_Unknown_NotFound(t *testing.T) {
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
	_ = svc.SetGauge(ctx, "HeapAlloc", 1.25)
	_ = svc.AddCounter(ctx, models.PollCount, 4)
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
	_ = svc.AddCounter(ctx, models.PollCount, 1)
	_ = svc.AddCounter(ctx, "OtherCounter", 2)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "PollCount: 1") || !strings.Contains(body, "OtherCounter: 2") {
		t.Fatalf("expected both counters in body %q", body)
	}
}

func TestHandler_Ping_NoDB_InternalServerError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mux := newPingMux(nil)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestHandler_Ping_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mux := newPingMux(&mockDBChecker{err: nil})
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestHandler_Ping_DBError_InternalServerError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mux := newPingMux(&mockDBChecker{err: errors.New("connection refused")})
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
}
