package agent_test

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/agent"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

func TestUpdateGaugesFromMemStats_MapsMemStats(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))
	ms := &runtime.MemStats{
		Alloc:         101,
		HeapAlloc:     202,
		TotalAlloc:    303,
		NumGC:         4,
		GCCPUFraction: 0.5,
		BuckHashSys:   1,
		Frees:         2,
		GCSys:         3,
		HeapIdle:      4,
		HeapInuse:     5,
		HeapObjects:   6,
		HeapReleased:  7,
		HeapSys:       8,
		LastGC:        9,
		Lookups:       10,
		MCacheInuse:   11,
		MCacheSys:     12,
		MSpanInuse:    13,
		MSpanSys:      14,
		Mallocs:       15,
		NextGC:        16,
		NumForcedGC:   17,
		OtherSys:      18,
		PauseTotalNs:  19,
		StackInuse:    20,
		StackSys:      21,
		Sys:           22,
	}

	m := make(map[string]float64)
	models.UpdateGaugesFromMemStats(m, ms, rnd)

	if m["Alloc"] != 101 {
		t.Fatalf("Alloc: got %v", m["Alloc"])
	}
	if m["HeapAlloc"] != 202 {
		t.Fatalf("HeapAlloc: got %v", m["HeapAlloc"])
	}
	if m["GCCPUFraction"] != 0.5 {
		t.Fatalf("GCCPUFraction: got %v", m["GCCPUFraction"])
	}

	rv := m["RandomValue"]
	matched := false
	for _, n := range models.GaugeNames {
		if m[n] == rv {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("RandomValue %v is not equal to any listed gauge field", rv)
	}
}

func TestPostMetric_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Content-Encoding %q", r.Header.Get("Content-Encoding"))
		}
		if r.URL.Path != "/update" {
			t.Errorf("path %s", r.URL.Path)
		}
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		b, _ := io.ReadAll(zr)
		_ = zr.Close()
		var m models.Metrics
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("body: %v", err)
		}
		if m.ID != "foo" || m.MType != models.Gauge || m.Value == nil || *m.Value != 1.5 || m.Delta != nil {
			t.Fatalf("metric %+v", m)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := agent.PostGaugeMetric(client, "", srv.URL, "foo", models.Gauge, 1.5, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostMetricsBatch_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Content-Encoding %q", r.Header.Get("Content-Encoding"))
		}
		if r.URL.Path != "/updates/" {
			t.Errorf("path %s", r.URL.Path)
		}
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		b, _ := io.ReadAll(zr)
		_ = zr.Close()
		var ms []models.Metrics
		if err := json.Unmarshal(b, &ms); err != nil {
			t.Fatalf("body: %v", err)
		}
		if len(ms) != 2 {
			t.Fatalf("metrics len %d: %+v", len(ms), ms)
		}
		if ms[0].ID != "foo" || ms[0].MType != models.Gauge || ms[0].Value == nil || *ms[0].Value != 1.5 || ms[0].Delta != nil {
			t.Fatalf("metric[0] %+v", ms[0])
		}
		if ms[1].ID != models.PollCount || ms[1].MType != models.Counter || ms[1].Value != nil || ms[1].Delta == nil || *ms[1].Delta != 7 {
			t.Fatalf("metric[1] %+v", ms[1])
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	v := 1.5
	d := int64(7)
	err := agent.PostMetricsBatch(client, "", srv.URL, []models.Metrics{
		{ID: "foo", MType: models.Gauge, Value: &v},
		{ID: models.PollCount, MType: models.Counter, Delta: &d},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostMetricsBatch_Empty_NoRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	if err := agent.PostMetricsBatch(client, "", srv.URL, nil, nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if called {
		t.Fatal("expected no request for empty batch")
	}
}

func TestPostMetric_NaN(t *testing.T) {
	client := resty.New()
	err := agent.PostGaugeMetric(client, "", "http://unused", "x", models.Gauge, math.NaN(), nil)
	if err == nil {
		t.Fatal("expected error for NaN")
	}
}

func TestPostMetric_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := agent.PostGaugeMetric(client, "", srv.URL, "a", models.Gauge, 1, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPostIntMetric_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Content-Encoding %q", r.Header.Get("Content-Encoding"))
		}
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip reader: %v", err)
		}
		b, _ := io.ReadAll(zr)
		_ = zr.Close()
		var m models.Metrics
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("body: %v", err)
		}
		if m.ID != models.PollCount || m.MType != models.Counter || m.Value != nil || m.Delta == nil || *m.Delta != 7 {
			t.Fatalf("metric %+v", m)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := agent.PostCounterMetric(client, "", srv.URL, models.PollCount, models.Counter, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
}
