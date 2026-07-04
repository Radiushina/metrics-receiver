package repository_test

import (
	"context"
	"testing"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

// benchAgentBatch формирует пакет метрик как при отправке с агента:
// все gauge из runtime.MemStats + счётчики PollCount.
func benchAgentBatch() []models.Metrics {
	metrics := make([]models.Metrics, 0, len(models.GaugeNames)+len(models.CounterNames))
	for _, name := range models.GaugeNames {
		v := 1.0
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &v,
		})
	}
	for _, name := range models.CounterNames {
		d := int64(1)
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &d,
		})
	}
	return metrics
}

// BenchmarkMemoryRepoSetGauge измеряет скорость записи одного gauge в in-memory репозиторий.
func BenchmarkMemoryRepoSetGauge(b *testing.B) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = repo.SetGauge(ctx, "HeapAlloc", 42.5)
	}
}

// BenchmarkMemoryRepoAddCounter измеряет скорость инкремента одного counter.
func BenchmarkMemoryRepoAddCounter(b *testing.B) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = repo.AddCounter(ctx, models.PollCount, 1)
	}
}

// BenchmarkMemoryRepoUpdateMetricsBatch измеряет пакетное обновление метрик —
// основной сценарий при POST /updates/ от агента.
func BenchmarkMemoryRepoUpdateMetricsBatch(b *testing.B) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()
	batch := benchAgentBatch()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = repo.UpdateMetricsBatch(ctx, batch)
	}
}

// BenchmarkMemoryRepoGauges измеряет чтение всех gauge (копия map для GET /).
func BenchmarkMemoryRepoGauges(b *testing.B) {
	ctx := context.Background()
	repo := repository.NewMemoryRepo()
	batch := benchAgentBatch()
	_ = repo.UpdateMetricsBatch(ctx, batch)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = repo.Gauges(ctx)
	}
}
