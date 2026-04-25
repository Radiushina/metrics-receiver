package models

// MetricType is a metric kind supported by the service.
type MetricType string

// Supported metric kinds and well-known metric names.
const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	PollCount            = "PollCount"
)

// Metrics represents a single metric in the JSON API.
//
// Delta and Value are pointers to distinguish an explicit 0 from an absent field.
type Metrics struct {
	ID    string     `json:"id"`
	MType MetricType `json:"type"`
	Delta *int64     `json:"delta,omitempty"`
	Value *float64   `json:"value,omitempty"`
}
