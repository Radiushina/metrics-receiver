package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"github.com/Radiushina/metrics-receiver.git/internal/model"
	"go.uber.org/zap"
)

type metricRow struct {
	Name  string
	Value string
}

type metricsIndexData struct {
	Gauges   []metricRow
	Counters []metricRow
}

var metricsIndexTmpl = template.Must(template.New("metricsIndex").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Metrics</title>
</head>
<body>
<h1>Metrics</h1>
{{if .Gauges}}
<h2>Gauges</h2>
<ul>
{{range .Gauges}}<li>{{.Name}}: {{.Value}}</li>
{{end}}</ul>
{{end}}
{{if .Counters}}
<h2>Counters</h2>
<ul>
{{range .Counters}}<li>{{.Name}}: {{.Value}}</li>
{{end}}</ul>
{{end}}
{{if and (not .Gauges) (not .Counters)}}<p>No metrics yet.</p>{{end}}
</body>
</html>
`))

type (
	Handler struct {
		service ServiceProvider
	}

	ServiceProvider interface {
		SetGauge(name string, value float64)
		AddCounter(name string, delta int64)
		GetGauge(name string) (float64, bool)
		GetCounter(name string) (int64, bool)
		Gauges() map[string]float64
		Counters() map[string]int64
	}
)

func NewHandler(service ServiceProvider) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetricsValue(w, h.service)
	}
}

func (h *Handler) GetMetricValue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetric(w, r, h.service)
	}
}

func (h *Handler) UpdateMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetrics(w, r, h.service)
	}
}

func updateMetrics(w http.ResponseWriter, r *http.Request, service ServiceProvider) {
	defer func() { _ = r.Body.Close() }()

	var metrics models.Metrics
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	logger.Log.Info("metrics update request body", zap.String("body", string(buf.Bytes())))

	if err := json.Unmarshal(buf.Bytes(), &metrics); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if metrics.ID == "" {
		http.Error(w, "missing metric id", http.StatusBadRequest)
		return
	}

	mtype := models.MetricType(strings.ToLower(string(metrics.MType)))

	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	switch mtype {
	case models.Counter:
		if metrics.Delta == nil {
			http.Error(w, "missing counter delta", http.StatusBadRequest)
			return
		}
		service.AddCounter(metrics.ID, *metrics.Delta)
		logger.Log.Sugar().Infof("server: counter %s += %d", metrics.ID, *metrics.Delta)
	case models.Gauge:
		if metrics.Value == nil {
			http.Error(w, "missing gauge value", http.StatusBadRequest)
			return
		}
		service.SetGauge(metrics.ID, *metrics.Value)
		logger.Log.Sugar().Infof("server: gauge %s = %g", metrics.ID, *metrics.Value)
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func getMetric(w http.ResponseWriter, r *http.Request, service ServiceProvider) {
	defer func() { _ = r.Body.Close() }()

	var req models.Metrics
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	logger.Log.Info("metric value request body", zap.String("body", string(buf.Bytes())))

	if err := json.Unmarshal(buf.Bytes(), &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "missing metric id", http.StatusBadRequest)
		return
	}

	mtype := models.MetricType(strings.ToLower(string(req.MType)))
	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	out := models.Metrics{
		ID:    req.ID,
		MType: mtype,
	}

	switch mtype {
	case models.Gauge:
		v, ok := service.GetGauge(req.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		out.Value = &v
	case models.Counter:
		v, ok := service.GetCounter(req.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		out.Delta = &v
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
		return
	}

	respBody, err := json.Marshal(out)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func getMetricsValue(w http.ResponseWriter, service ServiceProvider) {
	gauges := service.Gauges()

	gNames := make([]string, 0, len(gauges))
	for n := range gauges {
		gNames = append(gNames, n)
	}
	sort.Strings(gNames)
	gRows := make([]metricRow, 0, len(gNames))
	for _, n := range gNames {
		gRows = append(gRows, metricRow{
			Name:  n,
			Value: strconv.FormatFloat(gauges[n], 'g', -1, 64),
		})
	}

	counters := service.Counters()
	cNames := make([]string, 0, len(counters))
	for n := range counters {
		cNames = append(cNames, n)
	}
	sort.Strings(cNames)
	cRows := make([]metricRow, 0, len(cNames))
	for _, n := range cNames {
		cRows = append(cRows, metricRow{
			Name:  n,
			Value: strconv.FormatInt(counters[n], 10),
		})
	}

	var buf bytes.Buffer
	data := metricsIndexData{Gauges: gRows, Counters: cRows}
	if err := metricsIndexTmpl.Execute(&buf, data); err != nil {
		logger.Log.Error("metrics index template", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
