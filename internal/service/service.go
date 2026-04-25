package service

type (
	// Service provides a thin API over the metrics repository.
	Service struct {
		repo RepositoryProvider
	}

	// RepositoryProvider describes the repository operations.
	RepositoryProvider interface {
		SetGauge(name string, value float64)
		AddCounter(name string, delta int64)
		GetGauge(name string) (float64, bool)
		GetCounter(name string) (int64, bool)
		Gauges() map[string]float64
		Counters() map[string]int64
	}
)

// NewService constructs a Service backed by the provided repository.
func NewService(repo RepositoryProvider) Service {
	return Service{
		repo: repo,
	}
}

// SetGauge stores a gauge value by name.
func (s Service) SetGauge(name string, value float64) {
	s.repo.SetGauge(name, value)
}

// AddCounter increments a counter by delta.
func (s Service) AddCounter(name string, value int64) {
	s.repo.AddCounter(name, value)
}

// GetGauge returns a gauge value and whether it exists.
func (s Service) GetGauge(name string) (float64, bool) {
	return s.repo.GetGauge(name)
}

// GetCounter returns a counter value and whether it exists.
func (s Service) GetCounter(name string) (int64, bool) {
	return s.repo.GetCounter(name)
}

// Gauges returns a copy of all gauge values.
func (s Service) Gauges() map[string]float64 {
	return s.repo.Gauges()
}

// Counters returns a copy of all counter values.
func (s Service) Counters() map[string]int64 {
	return s.repo.Counters()
}
