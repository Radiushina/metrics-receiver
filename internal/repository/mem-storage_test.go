package repository

import (
	"testing"
)

func TestMemStorage_SetGauge_Replaces(t *testing.T) {
	s := NewMemStorage()
	s.SetGauge("HeapAlloc", 10)
	s.SetGauge("HeapAlloc", 20)

	if s.gauges["HeapAlloc"] != 20 {
		t.Fatalf("expected gauge 20, got %v", s.gauges["HeapAlloc"])
	}
}

func TestMemStorage_AddCounter_Accumulates(t *testing.T) {
	s := NewMemStorage()
	s.AddCounter("PollCount", 1)
	s.AddCounter("PollCount", 5)

	if s.counters["PollCount"] != 6 {
		t.Fatalf("expected counter 6, got %v", s.counters["PollCount"])
	}
}

func TestMemStorage_GaugeAndCounter_Independent(t *testing.T) {
	s := NewMemStorage()
	s.SetGauge("x", 1.5)
	s.AddCounter("x", 3)

	if s.gauges["x"] != 1.5 {
		t.Fatalf("gauge: %v", s.gauges["x"])
	}
	if s.counters["x"] != 3 {
		t.Fatalf("counter: %v", s.counters["x"])
	}
}
