package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

// PostMetric отправляет на сервер значение метрики типа gauge.
func PostMetric(
	client *resty.Client,
	baseURL, name string,
	metricType models.MetricType,
	value float64,
) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("invalid float value: %v", value)
	}
	return basePostMetric(client, baseURL, name, metricType, &value, nil)
}

// PostIntMetric отправляет на сервер приращение (delta) метрики типа counter.
func PostIntMetric(
	client *resty.Client,
	baseURL, name string,
	metricType models.MetricType,
	delta int64,
) error {
	return basePostMetric(client, baseURL, name, metricType, nil, &delta)
}

// PostMetricsBatch отправляет пакет метрик на сервер одним запросом.
// Тело запроса — JSON-массив models.Metrics, сжатый gzip, отправляется методом POST на /updates/.
func PostMetricsBatch(
	client *resty.Client,
	baseURL string,
	metrics []models.Metrics,
) error {
	if len(metrics) == 0 {
		return nil
	}
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil || m.Delta != nil {
				return errors.New("gauge metric requires value, delta must be omitted")
			}
			if math.IsNaN(*m.Value) || math.IsInf(*m.Value, 0) {
				return fmt.Errorf("invalid float value: %v", *m.Value)
			}
		case models.Counter:
			if m.Delta == nil || m.Value != nil {
				return errors.New("counter metric requires delta, value must be omitted")
			}
		default:
			return fmt.Errorf("unsupported metric type: %q", m.MType)
		}
		if strings.TrimSpace(m.ID) == "" {
			return errors.New("missing metric id")
		}
	}

	body, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	gzBody, err := gzipBytes(body)
	if err != nil {
		return err
	}
	return postGzippedMetricsBatch(client, baseURL, gzBody)
}

func basePostMetric(
	client *resty.Client,
	baseURL, name string,
	metricType models.MetricType,
	value *float64,
	delta *int64,
) error {
	if err := validateMetricArgs(metricType, value, delta); err != nil {
		return err
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

	gzBody, err := gzipBytes(body)
	if err != nil {
		return err
	}

	return postGzippedMetric(client, baseURL, gzBody)
}

func validateMetricArgs(
	metricType models.MetricType,
	value *float64,
	delta *int64,
) error {
	switch metricType {
	case models.Gauge:
		if value == nil || delta != nil {
			return errors.New("gauge metric requires value, delta must be omitted")
		}
		return nil
	case models.Counter:
		if delta == nil || value != nil {
			return errors.New("counter metric requires delta, value must be omitted")
		}
		return nil
	default:
		return fmt.Errorf("unsupported metric type: %q", metricType)
	}
}

func postGzippedMetric(client *resty.Client, baseURL string, gzBody []byte) error {
	fullURL, err := url.JoinPath(baseURL, "update")
	if err != nil {
		return err
	}

	return retryAgent(func() (bool, bool, error) {
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(gzBody).
			Post(fullURL)
		if err != nil {
			return false, isRetriableTransportErr(err), err
		}
		if resp.StatusCode() == http.StatusOK {
			return true, false, nil
		}
		if isRetriableHTTPStatus(resp.StatusCode()) {
			return false, true, fmt.Errorf("unexpected status %s", resp.Status())
		}
		return false, false, fmt.Errorf("unexpected status %s", resp.Status())
	})
}

func postGzippedMetricsBatch(client *resty.Client, baseURL string, gzBody []byte) error {
	fullURL := strings.TrimRight(baseURL, "/") + "/updates/"

	return retryAgent(func() (bool, bool, error) {
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("Accept-Encoding", "gzip").
			SetBody(gzBody).
			Post(fullURL)
		if err != nil {
			return false, isRetriableTransportErr(err), err
		}
		if resp.StatusCode() == http.StatusOK {
			return true, false, nil
		}
		if isRetriableHTTPStatus(resp.StatusCode()) {
			return false, true, fmt.Errorf("unexpected status %s", resp.Status())
		}
		return false, false, fmt.Errorf("unexpected status %s", resp.Status())
	})
}

func gzipBytes(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
