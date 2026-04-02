package service

type (
	Service struct {
		repo RepositoryProvider
	}

	RepositoryProvider interface {
		SetGauge(name string, value float64)
		AddCounter(name string, delta int64)
		GetGauge(name string) (float64, bool)
		GetCounter(name string) (int64, bool)
		Gauges() map[string]float64
		Counters() map[string]int64
	}
)

func NewService(repo RepositoryProvider) Service {
	return Service{
		repo: repo,
	}
}

func (s Service) SetGauge(name string, value float64) {
	s.repo.SetGauge(name, value)
}

func (s Service) AddCounter(name string, value int64) {
	s.repo.AddCounter(name, value)
}

func (s Service) GetGauge(name string) (float64, bool) {
	return s.repo.GetGauge(name)
}

func (s Service) GetCounter(name string) (int64, bool) {
	return s.repo.GetCounter(name)
}

func (s Service) Gauges() map[string]float64 {
	return s.repo.Gauges()
}

func (s Service) Counters() map[string]int64 {
	return s.repo.Counters()
}
