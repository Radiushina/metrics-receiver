package repository_test

import (
	"testing"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

func TestMemStorage_SetGauge_Replaces(t *testing.T) {
	s := repository.NewRepository()
	s.SetGauge("HeapAlloc", 10)
	s.SetGauge("HeapAlloc", 20)

	if got := s.Gauges()["HeapAlloc"]; got != 20 {
		t.Fatalf("expected gauge 20, got %v", got)
	}
}

func TestMemStorage_AddCounter_Accumulates(t *testing.T) {
	s := repository.NewRepository()
	s.AddCounter(models.PollCount, 1)
	s.AddCounter(models.PollCount, 5)

	if got := s.Counters()[models.PollCount]; got != 6 {
		t.Fatalf("expected counter 6, got %v", got)
	}
}

func TestMemStorage_SetCounter_Replaces(t *testing.T) {
	s := repository.NewRepository()
	s.AddCounter(models.PollCount, 1)
	s.SetCounter(models.PollCount, 42)
	s.AddCounter(models.PollCount, 5)

	if got := s.Counters()[models.PollCount]; got != 47 {
		t.Fatalf("expected counter 47, got %v", got)
	}
}

func TestMemStorage_GaugeAndCounter_Independent(t *testing.T) {
	s := repository.NewRepository()
	s.SetGauge("x", 1.5)
	s.AddCounter("x", 3)

	if got := s.Gauges()["x"]; got != 1.5 {
		t.Fatalf("gauge: %v", got)
	}
	if got := s.Counters()["x"]; got != 3 {
		t.Fatalf("counter: %v", got)
	}
}
