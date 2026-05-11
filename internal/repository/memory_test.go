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
	_ = s.SetGauge(ctx, "HeapAlloc", 10)
	_ = s.SetGauge(ctx, "HeapAlloc", 20)

	if got := s.Gauges(ctx)["HeapAlloc"]; got != 20 {
		t.Fatalf("expected gauge 20, got %v", got)
	}
}

func TestMemStorage_AddCounter_Accumulates(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	_ = s.AddCounter(ctx, models.PollCount, 1)
	_ = s.AddCounter(ctx, models.PollCount, 5)

	if got := s.Counters(ctx)[models.PollCount]; got != 6 {
		t.Fatalf("expected counter 6, got %v", got)
	}
}

func TestMemStorage_SetCounter_Replaces(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	_ = s.AddCounter(ctx, models.PollCount, 1)
	s.SetCounter(models.PollCount, 42)
	_ = s.AddCounter(ctx, models.PollCount, 5)

	if got := s.Counters(ctx)[models.PollCount]; got != 47 {
		t.Fatalf("expected counter 47, got %v", got)
	}
}

func TestMemStorage_GaugeAndCounter_Independent(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()
	_ = s.SetGauge(ctx, "x", 1.5)
	_ = s.AddCounter(ctx, "x", 3)

	if got := s.Gauges(ctx)["x"]; got != 1.5 {
		t.Fatalf("gauge: %v", got)
	}
	if got := s.Counters(ctx)["x"]; got != 3 {
		t.Fatalf("counter: %v", got)
	}
}

func TestMemStorage_UpdateMetricsBatch_AppliesAll(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemoryRepo()

	_ = s.SetGauge(ctx, "RandomValue", 1.0)
	_ = s.AddCounter(ctx, models.PollCount, 10)

	g1 := 3.14159
	g2 := 1547.0
	delta := int64(42)

	in := []models.Metrics{
		{ID: models.PollCount, MType: models.Counter, Delta: &delta},
		{ID: "RandomValue", MType: models.Gauge, Value: &g1},
		{ID: "ActiveUsers", MType: models.Gauge, Value: &g2},
	}

	if err := s.UpdateMetricsBatch(ctx, in); err != nil {
		t.Fatalf("UpdateMetricsBatch: %v", err)
	}

	if got := s.Counters(ctx)[models.PollCount]; got != 52 {
		t.Fatalf("counter: expected 52, got %v", got)
	}
	if got := s.Gauges(ctx)["RandomValue"]; got != g1 {
		t.Fatalf("gauge RandomValue: expected %v, got %v", g1, got)
	}
	if got := s.Gauges(ctx)["ActiveUsers"]; got != g2 {
		t.Fatalf("gauge ActiveUsers: expected %v, got %v", g2, got)
	}
}
