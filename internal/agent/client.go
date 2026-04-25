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

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-resty/resty/v2"
)

// PostMetric sends a gauge metric value to the server.
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

// PostIntMetric sends a counter metric delta to the server.
func PostIntMetric(
	client *resty.Client,
	baseURL, name string,
	metricType models.MetricType,
	delta int64,
) error {
	return basePostMetric(client, baseURL, name, metricType, nil, &delta)
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

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(gzBody).
		Post(fullURL)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status())
	}
	return nil
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
