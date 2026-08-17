// Package models определяет типы метрик и имена gauge/counter для API и агента.
package models

// MetricType — строковый перечислимый тип: вид метрики в HTTP/JSON API.
// Значение "counter" или "gauge" задаёт семантику полей Delta и Value.
type MetricType string

const (
	// Counter — счётчик: монотонно накапливаемое целое; в запросах передаётся приращение (delta).
	Counter MetricType = "counter"
	// Gauge — измеритель: произвольное вещественное значение в момент времени.
	Gauge MetricType = "gauge"
)

//go:generate go run ../../cmd/reset

// Metrics — одна метрика в теле JSON (обновление или запрос значения).
// generate:reset
type Metrics struct {
	// ID — имя метрики (строковый идентификатор).
	ID string `json:"id"`
	// MType — вид: counter или gauge; определяет, какое из полей Delta/Value используется.
	MType MetricType `json:"type"`
	// Delta — для counter: приращение или текущее значение в ответе API; nil — поле не передано.
	Delta *int64 `json:"delta,omitempty"`
	// Value — для gauge: значение с плавающей точкой; nil — поле не передано.
	Value *float64 `json:"value,omitempty"`
}
