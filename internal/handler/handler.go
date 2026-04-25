package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-chi/chi/v5"
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

var metricsIndexTmpl = template.Must(template.New("metricsIndex").
	Parse(`<!DOCTYPE html>
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
	// Handler wires HTTP handlers to the service layer.
	Handler struct {
		service ServiceProvider
		saver   Saver
	}

	// ServiceProvider describes the service operations required by Handler.
	ServiceProvider interface {
		SetGauge(name string, value float64)
		AddCounter(name string, delta int64)
		GetGauge(name string) (float64, bool)
		GetCounter(name string) (int64, bool)
		Gauges() map[string]float64
		Counters() map[string]int64
	}

	// Saver persists metrics after updates when enabled.
	Saver interface {
		Save(ctx context.Context) error
	}
)

// NewHandler constructs a Handler with the given service and optional saver.
func NewHandler(service ServiceProvider, saver Saver) *Handler {
	return &Handler{
		service: service,
		saver:   saver,
	}
}

// GetMetrics returns an HTTP handler
// that writes all collected metrics to the response.
func (h *Handler) GetMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeMetricsIndex(w, h.service)
	}
}

// GetMetric returns an HTTP handler
// that writes a single metric value in plain text format.
func (h *Handler) GetMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetricValue(w, r, h.service)
	}
}

// GetMetricValue returns an HTTP handler
// that writes a single metric value in JSON format.
func (h *Handler) GetMetricValue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetric(w, r, h.service)
	}
}

// UpdateFromPath returns an HTTP handler
// that updates a metric from URL path parameters.
func (h *Handler) UpdateFromPath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetricsFromPath(w, r, h.service, h.saver)
	}
}

// UpdateFromBody returns an HTTP handler
// that updates a metric from a JSON request body.
func (h *Handler) UpdateFromBody() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetricsFromBody(w, r, h.service, h.saver)
	}
}

func updateMetricsFromPath(w http.ResponseWriter, r *http.Request, service ServiceProvider, saver Saver) {
	mtype := models.MetricType(strings.ToLower(chi.URLParam(r, "mtype")))
	metric := chi.URLParam(r, "metric")
	valueStr := chi.URLParam(r, "value")

	if metric == "" {
		http.NotFound(w, r)
		return
	}

	if valueStr == "" {
		http.Error(w, "missing metric value", http.StatusBadRequest)
		return
	}

	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	switch mtype {
	case models.Gauge:
		v, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
		service.SetGauge(metric, v)
		logger.Log.Sugar().Infof("server: gauge %s = %g", metric, v)
	case models.Counter:
		v, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
		service.AddCounter(metric, v)
		logger.Log.Sugar().Infof("server: counter %s += %d", metric, v)
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
		return
	}

	if saver != nil {
		if err := saver.Save(r.Context()); err != nil {
			http.Error(w, "failed to persist metrics", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func updateMetricsFromBody(
	w http.ResponseWriter,
	r *http.Request,
	service ServiceProvider,
	saver Saver,
) {
	defer func() { _ = r.Body.Close() }()

	reqBody, err := readRequestBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	logRequestBody("metrics update request body", reqBody)

	in, err := unmarshalMetric(reqBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, status, err := applyMetricUpdate(service, in)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	if err := persistIfEnabled(r.Context(), saver); err != nil {
		http.Error(w, "failed to persist metrics", http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}

func readRequestBody(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("failed to read body")
	}
	return b, nil
}

func logRequestBody(msg string, b []byte) {
	logger.Log.Info(msg, zap.String("body", string(b)))
}

func unmarshalMetric(b []byte) (models.Metrics, error) {
	var m models.Metrics
	if err := json.Unmarshal(b, &m); err != nil {
		return models.Metrics{}, errors.New("invalid JSON")
	}
	if strings.TrimSpace(m.ID) == "" {
		return models.Metrics{}, errors.New("missing metric id")
	}
	m.MType = models.MetricType(strings.ToLower(string(m.MType)))
	if m.MType == "" {
		return models.Metrics{}, errors.New("missing metric type")
	}
	return m, nil
}

func applyMetricUpdate(service ServiceProvider, in models.Metrics) (models.Metrics, int, error) {
	switch in.MType {
	case models.Counter:
		return applyCounterUpdate(service, in)
	case models.Gauge:
		return applyGaugeUpdate(service, in)
	default:
		return models.Metrics{}, http.StatusBadRequest, fmt.Errorf("invalid metric type: %q", in.MType)
	}
}

func applyCounterUpdate(service ServiceProvider, in models.Metrics) (models.Metrics, int, error) {
	if in.Delta == nil {
		return models.Metrics{}, http.StatusBadRequest, errors.New("missing counter delta")
	}

	service.AddCounter(in.ID, *in.Delta)
	logger.Log.Sugar().Infof("server: counter %s += %d", in.ID, *in.Delta)

	total, ok := service.GetCounter(in.ID)
	if !ok {
		return models.Metrics{}, http.StatusInternalServerError, errors.New("internal server error")
	}

	return models.Metrics{
		ID:    in.ID,
		MType: models.Counter,
		Delta: &total,
	}, http.StatusOK, nil
}

func applyGaugeUpdate(service ServiceProvider, in models.Metrics) (models.Metrics, int, error) {
	if in.Value == nil {
		return models.Metrics{}, http.StatusBadRequest, errors.New("missing gauge value")
	}

	service.SetGauge(in.ID, *in.Value)
	logger.Log.Sugar().Infof("server: gauge %s = %g", in.ID, *in.Value)

	v := *in.Value
	return models.Metrics{
		ID:    in.ID,
		MType: models.Gauge,
		Value: &v,
	}, http.StatusOK, nil
}

func persistIfEnabled(ctx context.Context, saver Saver) error {
	if saver == nil {
		return nil
	}
	return saver.Save(ctx)
}

func writeJSON(w http.ResponseWriter, v any) {
	respBody, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func getMetric(w http.ResponseWriter, r *http.Request, service ServiceProvider) {
	defer func() { _ = r.Body.Close() }()

	var req models.Metrics
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	logger.Log.Info("metric value request body",
		zap.String("body", buf.String()))

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

func getMetricValue(w http.ResponseWriter, r *http.Request, service ServiceProvider) {
	mtype := models.MetricType(strings.ToLower(chi.URLParam(r, "mtype")))
	metric := chi.URLParam(r, "metric")

	if metric == "" {
		http.NotFound(w, r)
		return
	}

	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	switch mtype {
	case models.Gauge:
		v, ok := service.GetGauge(metric)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatFloat(v, 'g', -1, 64)))
	case models.Counter:
		v, ok := service.GetCounter(metric)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(v, 10)))
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
	}
}

func writeMetricsIndex(w http.ResponseWriter, service ServiceProvider) {
	gauges := service.Gauges()

	gNames := make([]string, 0, len(gauges))
	for n := range gauges {
		gNames = append(gNames, n)
	}
	slices.Sort(gNames)
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
	slices.Sort(cNames)
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
