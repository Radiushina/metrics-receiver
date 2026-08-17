package proto

import (
	"errors"
	"fmt"
	"strings"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

// RealIPMetadataKey — ключ gRPC-метаданных с IP-адресом агента (канонический lowercase).
const RealIPMetadataKey = "x-real-ip"

// ToProto конвертирует доменные метрики в protobuf.
func ToProto(metrics []models.Metrics) []*Metric {
	out := make([]*Metric, 0, len(metrics))
	for _, m := range metrics {
		pm := &Metric{Id: m.ID}
		switch m.MType {
		case models.Gauge:
			pm.Type = Metric_GAUGE
			if m.Value != nil {
				pm.Value = *m.Value
			}
		case models.Counter:
			pm.Type = Metric_COUNTER
			if m.Delta != nil {
				pm.Delta = *m.Delta
			}
		default:
			continue
		}
		out = append(out, pm)
	}
	return out
}

// FromProto конвертирует protobuf-метрики в доменную модель.
func FromProto(in []*Metric) ([]models.Metrics, error) {
	out := make([]models.Metrics, 0, len(in))
	for _, m := range in {
		if m == nil {
			continue
		}
		if strings.TrimSpace(m.GetId()) == "" {
			return nil, errors.New("missing metric id")
		}
		switch m.GetType() {
		case Metric_GAUGE:
			v := m.GetValue()
			out = append(out, models.Metrics{ID: m.GetId(), MType: models.Gauge, Value: &v})
		case Metric_COUNTER:
			d := m.GetDelta()
			out = append(out, models.Metrics{ID: m.GetId(), MType: models.Counter, Delta: &d})
		default:
			return nil, fmt.Errorf("invalid metric type: %v", m.GetType())
		}
	}
	return out, nil
}
