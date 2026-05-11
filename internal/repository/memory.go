package repository

import (
	"context"
	"sync"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

// MemoryRepo — хранилище метрик в оперативной памяти, потокобезопасное.
type MemoryRepo struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemoryRepo возвращает пустой репозиторий в памяти.
func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// SetGauge записывает значение gauge по имени метрики.
func (r *MemoryRepo) SetGauge(_ context.Context, name string, value float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
	return nil
}

// AddCounter увеличивает счётчик на delta.
func (r *MemoryRepo) AddCounter(_ context.Context, name string, delta int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += delta
	return nil
}

// SetCounter задаёт абсолютное значение счётчика по имени.
func (r *MemoryRepo) SetCounter(name string, value int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] = value
}

// GetGauge возвращает значение gauge и признак, что метрика есть.
func (r *MemoryRepo) GetGauge(_ context.Context, name string) (float64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.gauges[name]
	return v, ok
}

// GetCounter возвращает значение счётчика и признак, что метрика есть.
func (r *MemoryRepo) GetCounter(_ context.Context, name string) (int64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.counters[name]
	return v, ok
}

// Gauges возвращает копию всех gauge-значений.
func (r *MemoryRepo) Gauges(_ context.Context) map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]float64, len(r.gauges))
	for k, v := range r.gauges {
		out[k] = v
	}
	return out
}

// Counters возвращает копию всех значений счётчиков.
func (r *MemoryRepo) Counters(_ context.Context) map[string]int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]int64, len(r.counters))
	for k, v := range r.counters {
		out[k] = v
	}
	return out
}

// UpdateMetricsBatch применяет список обновлений метрик атомарно относительно других операций репозитория.
// Для gauge выполняется запись абсолютного значения, для counter — инкремент на delta.
// В отличие от Postgres-репозитория, здесь нет транзакций БД, но обновление защищено mutex'ом:
// другие операции не увидят частично применённый список.
func (r *MemoryRepo) UpdateMetricsBatch(_ context.Context, metrics []models.Metrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			r.gauges[m.ID] = *m.Value
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			r.counters[m.ID] += *m.Delta
		}
	}
	return nil
}
