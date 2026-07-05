package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

func Example() {
	// Создаём handler на in-memory сервисе и регистрируем маршруты практического трека.
	ctx := context.Background()
	svc := newMockService()
	mux := newTestMux(svc)

	// POST /update — обновление одной метрики из JSON.
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge","value":42.5}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// {"id":"HeapAlloc","type":"gauge","value":42.5}
}

func ExampleHandler_UpdateFromBody() {
	ctx := context.Background()
	mux := newTestMux(newMockService())

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update", strings.NewReader(
		`{"id":"PollCount","type":"counter","delta":3}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// {"id":"PollCount","type":"counter","delta":3}
}

func ExampleHandler_UpdateFromPath() {
	ctx := context.Background()
	mux := newTestMux(newMockService())

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/update/gauge/HeapAlloc/42.5", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	//
}

func ExampleHandler_UpdateMetrics() {
	ctx := context.Background()
	mux := newTestMux(newMockService())

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/updates/", strings.NewReader(
		`[
  {"id":"PollCount","type":"counter","delta":1},
  {"id":"HeapAlloc","type":"gauge","value":42.5}
]`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// [{"id":"PollCount","type":"counter","delta":1},{"id":"HeapAlloc","type":"gauge","value":42.5}]
}

func ExampleHandler_GetMetric() {
	ctx := context.Background()
	svc := newMockService()
	_ = svc.SetGauge(ctx, "HeapAlloc", 42.5)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/value/gauge/HeapAlloc", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// 42.5
}

func ExampleHandler_GetMetricValue() {
	ctx := context.Background()
	svc := newMockService()
	_ = svc.AddCounter(ctx, models.PollCount, 7)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/value", strings.NewReader(
		`{"id":"PollCount","type":"counter"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Body.String())

	// Output:
	// 200
	// {"id":"PollCount","type":"counter","delta":7}
}

func ExampleHandler_GetMetrics() {
	ctx := context.Background()
	svc := newMockService()
	_ = svc.SetGauge(ctx, "HeapAlloc", 1.25)
	mux := newTestMux(svc)

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(strings.Contains(rec.Body.String(), "HeapAlloc: 1.25"))

	// Output:
	// 200
	// true
}

func ExampleHandler_PingDB() {
	ctx := context.Background()
	mux := newPingMux(&mockDBChecker{err: nil})

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Code)

	// Output:
	// 200
}
