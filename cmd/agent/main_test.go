package main

import (
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

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
	updateGaugesFromMemStats(m, ms, rnd)

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
	for _, n := range gaugeNames {
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
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("Content-Type %q", r.Header.Get("Content-Type"))
		}
		if r.URL.Path != "/update/gauge/foo/1.5" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := postMetric(client, srv.URL, metricTypeGauge, "foo", 1.5)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostMetric_NaN(t *testing.T) {
	client := resty.New()
	err := postMetric(client, "http://unused", metricTypeGauge, "x", math.NaN())
	if err == nil {
		t.Fatal("expected error for NaN")
	}
}

func TestPostMetric_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := postMetric(client, srv.URL, metricTypeGauge, "a", 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPostIntMetric_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update/counter/PollCount/7" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := resty.New().SetTimeout(5 * time.Second)
	err := postIntMetric(client, srv.URL, metricTypeCounter, models.PollCount, 7)
	if err != nil {
		t.Fatal(err)
	}
}
