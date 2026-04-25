package repository

import "sync"

// Repository is an in-memory, concurrency-safe storage for metrics.
type Repository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewRepository returns an empty in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// SetGauge stores a gauge value by name.
func (r *Repository) SetGauge(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

// AddCounter increments a counter by delta.
func (r *Repository) AddCounter(name string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += delta
}

// SetCounter stores an absolute counter value by name.
func (r *Repository) SetCounter(name string, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] = value
}

// GetGauge returns a gauge value and whether it exists.
func (r *Repository) GetGauge(name string) (float64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.gauges[name]
	return v, ok
}

// GetCounter returns a counter value and whether it exists.
func (r *Repository) GetCounter(name string) (int64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.counters[name]
	return v, ok
}

// Gauges returns a copy of all gauge values.
func (r *Repository) Gauges() map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]float64, len(r.gauges))
	for k, v := range r.gauges {
		out[k] = v
	}
	return out
}

// Counters returns a copy of all counter values.
func (r *Repository) Counters() map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]int64, len(r.counters))
	for k, v := range r.counters {
		out[k] = v
	}
	return out
}
