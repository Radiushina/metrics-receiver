package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

func PostMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("invalid float value: %v", value)
	}
	return basePostMetric(client, baseURL, name, metricType, &value, nil)
}

func PostIntMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, delta int64) error {
	return basePostMetric(client, baseURL, name, metricType, nil, &delta)
}

func basePostMetric(client *resty.Client, baseURL, name string, metricType models.MetricType, value *float64, delta *int64) error {
	switch metricType {
	case models.Gauge:
		if value == nil || delta != nil {
			return fmt.Errorf("gauge metric requires value, delta must be omitted")
		}
	case models.Counter:
		if delta == nil || value != nil {
			return fmt.Errorf("counter metric requires delta, value must be omitted")
		}
	default:
		return fmt.Errorf("unsupported metric type: %q", metricType)
	}

	metric := models.Metrics{
		ID:    name,
		MType: metricType,
		Delta: delta,
		Value: value,
	}

	body, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	fullURL, err := url.JoinPath(baseURL, "update")
	if err != nil {
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(fullURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status())
	}
	return nil
}
