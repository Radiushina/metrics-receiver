// Package service предоставляет слой бизнес-логики над репозиторием метрик.
package service

import (
	"context"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

type (
	// Service — обёртка над RepositoryProvider.
	Service struct {
		repo RepositoryProvider
	}

	// RepositoryProvider описывает операции хранилища метрик.
	RepositoryProvider interface {
		SetGauge(ctx context.Context, name string, value float64) error
		AddCounter(ctx context.Context, name string, delta int64) error
		GetGauge(ctx context.Context, name string) (float64, bool)
		GetCounter(ctx context.Context, name string) (int64, bool)
		Gauges(ctx context.Context) map[string]float64
		Counters(ctx context.Context) map[string]int64
		UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	}
)

// NewService создаёт Service с указанным репозиторием.
func NewService(repo RepositoryProvider) Service {
	return Service{
		repo: repo,
	}
}

// SetGauge сохраняет значение gauge по имени метрики.
func (s Service) SetGauge(ctx context.Context, name string, value float64) error {
	return s.repo.SetGauge(ctx, name, value)
}

// AddCounter увеличивает счётчик на delta.
func (s Service) AddCounter(ctx context.Context, name string, value int64) error {
	return s.repo.AddCounter(ctx, name, value)
}

// GetGauge возвращает значение gauge и признак, что метрика есть.
func (s Service) GetGauge(ctx context.Context, name string) (float64, bool) {
	return s.repo.GetGauge(ctx, name)
}

// GetCounter возвращает значение счётчика и признак, что метрика есть.
func (s Service) GetCounter(ctx context.Context, name string) (int64, bool) {
	return s.repo.GetCounter(ctx, name)
}

// Gauges возвращает копию всех gauge-значений.
func (s Service) Gauges(ctx context.Context) map[string]float64 {
	return s.repo.Gauges(ctx)
}

// Counters возвращает копию всех значений счётчиков.
func (s Service) Counters(ctx context.Context) map[string]int64 {
	return s.repo.Counters(ctx)
}

// UpdateMetricsBatch применяет список обновлений метрик за одну операцию.
func (s Service) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	return s.repo.UpdateMetricsBatch(ctx, metrics)
}
