package service

import "context"

type (
	// Service provides a thin API over the metrics repository.
	Service struct {
		repo RepositoryProvider
	}

	// RepositoryProvider describes the repository operations.
	RepositoryProvider interface {
		SetGauge(ctx context.Context, name string, value float64)
		AddCounter(ctx context.Context, name string, delta int64)
		GetGauge(ctx context.Context, name string) (float64, bool)
		GetCounter(ctx context.Context, name string) (int64, bool)
		Gauges(ctx context.Context) map[string]float64
		Counters(ctx context.Context) map[string]int64
	}
)

// NewService constructs a Service backed by the provided repository.
func NewService(repo RepositoryProvider) Service {
	return Service{
		repo: repo,
	}
}

// SetGauge stores a gauge value by name.
func (s Service) SetGauge(ctx context.Context, name string, value float64) {
	s.repo.SetGauge(ctx, name, value)
}

// AddCounter increments a counter by delta.
func (s Service) AddCounter(ctx context.Context, name string, value int64) {
	s.repo.AddCounter(ctx, name, value)
}

// GetGauge returns a gauge value and whether it exists.
func (s Service) GetGauge(ctx context.Context, name string) (float64, bool) {
	return s.repo.GetGauge(ctx, name)
}

// GetCounter returns a counter value and whether it exists.
func (s Service) GetCounter(ctx context.Context, name string) (int64, bool) {
	return s.repo.GetCounter(ctx, name)
}

// Gauges returns a copy of all gauge values.
func (s Service) Gauges(ctx context.Context) map[string]float64 {
	return s.repo.Gauges(ctx)
}

// Counters returns a copy of all counter values.
func (s Service) Counters(ctx context.Context) map[string]int64 {
	return s.repo.Counters(ctx)
}
