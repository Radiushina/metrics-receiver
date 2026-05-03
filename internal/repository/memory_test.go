package repository_test

import (
	"context"
	"testing"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

func TestMemStorage_SetGauge_Replaces(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	s.SetGauge(ctx, "HeapAlloc", 10)
	s.SetGauge(ctx, "HeapAlloc", 20)

	if got := s.Gauges(ctx)["HeapAlloc"]; got != 20 {
		t.Fatalf("expected gauge 20, got %v", got)
	}
}

func TestMemStorage_AddCounter_Accumulates(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	s.AddCounter(ctx, models.PollCount, 1)
	s.AddCounter(ctx, models.PollCount, 5)

	if got := s.Counters(ctx)[models.PollCount]; got != 6 {
		t.Fatalf("expected counter 6, got %v", got)
	}
}

func TestMemStorage_SetCounter_Replaces(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	s.AddCounter(ctx, models.PollCount, 1)
	s.SetCounter(models.PollCount, 42)
	s.AddCounter(ctx, models.PollCount, 5)

	if got := s.Counters(ctx)[models.PollCount]; got != 47 {
		t.Fatalf("expected counter 47, got %v", got)
	}
}

func TestMemStorage_GaugeAndCounter_Independent(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	s.SetGauge(ctx, "x", 1.5)
	s.AddCounter(ctx, "x", 3)

	if got := s.Gauges(ctx)["x"]; got != 1.5 {
		t.Fatalf("gauge: %v", got)
	}
	if got := s.Counters(ctx)["x"]; got != 3 {
		t.Fatalf("counter: %v", got)
	}
}
