package repository

import "sync"

type MemRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewRepository() *MemRepository {
	return &MemRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemRepository) SetGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemRepository) AddCounter(name string, delta int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += delta
}

func (s *MemRepository) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	return v, ok
}

func (s *MemRepository) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	return v, ok
}

func (s *MemRepository) Gauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		out[k] = v
	}
	return out
}

func (s *MemRepository) Counters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		out[k] = v
	}
	return out
}
