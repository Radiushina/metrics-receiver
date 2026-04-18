package repository

import "sync"

type Repository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewRepository() *Repository {
	return &Repository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (r *Repository) SetGauge(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

func (r *Repository) AddCounter(name string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += delta
}

func (r *Repository) GetGauge(name string) (float64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.gauges[name]
	return v, ok
}

func (r *Repository) GetCounter(name string) (int64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.counters[name]
	return v, ok
}

func (r *Repository) Gauges() map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]float64, len(r.gauges))
	for k, v := range r.gauges {
		out[k] = v
	}
	return out
}

func (r *Repository) Counters() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]int64, len(r.counters))
	for k, v := range r.counters {
		out[k] = v
	}
	return out
}
