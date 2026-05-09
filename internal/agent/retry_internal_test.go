package agent

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

func TestRetry_PostMetric_TransportFailsThenOK(t *testing.T) {
	old := retrySleep
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { retrySleep = old })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip: %v", err)
		}
		_, _ = io.ReadAll(zr)
		_ = zr.Close()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	var failCount atomic.Int32

	tr := &flakyTransport{
		base: http.DefaultTransport,
		failFn: func() bool {
			// Три искусственных сбоя, затем успешный запрос к srv.
			if failCount.Load() < 3 {
				failCount.Add(1)
				return true
			}
			return false
		},
	}

	client := resty.New().
		SetTransport(tr).
		SetTimeout(5 * time.Second)

	err := PostMetric(client, srv.URL, "x", models.Gauge, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := int(tr.attempts.Load()); got != 4 {
		t.Fatalf("expected 4 HTTP attempts, got %d", got)
	}
}

func TestRetry_PostMetric_TransportAlwaysFails(t *testing.T) {
	old := retrySleep
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { retrySleep = old })

	tr := &flakyTransport{
		base: http.DefaultTransport,
		failFn: func() bool {
			return true
		},
	}

	client := resty.New().
		SetTransport(tr).
		SetTimeout(5 * time.Second)

	err := PostMetric(client, "http://127.0.0.1:1", "x", models.Gauge, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := int(tr.attempts.Load()); got != 4 {
		t.Fatalf("expected 4 attempts, got %d", got)
	}
}

func TestRetry_PostMetricsBatch_Status503ThenOK(t *testing.T) {
	old := retrySleep
	retrySleep = func(time.Duration) {}
	t.Cleanup(func() { retrySleep = old })

	var handlerCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := handlerCalls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip: %v", err)
		}
		_, _ = io.ReadAll(zr)
		_ = zr.Close()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	v := 1.0
	err := PostMetricsBatch(client, srv.URL, []models.Metrics{
		{ID: "a", MType: models.Gauge, Value: &v},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := handlerCalls.Load(); got != 3 {
		t.Fatalf("expected 3 handler invocations, got %d", got)
	}
}

type flakyTransport struct {
	base     http.RoundTripper
	failFn   func() bool
	attempts atomic.Int32
}

func (t *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.attempts.Add(1)
	if t.failFn != nil && t.failFn() {
		return nil, syscall.ECONNREFUSED
	}
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	return t.base.RoundTrip(req)
}
