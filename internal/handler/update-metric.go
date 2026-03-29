package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/go-chi/chi/v5"
)

type metricRow struct {
	Name  string
	Value string
}

type metricsIndexData struct {
	Gauges       []metricRow
	HasCounter   bool
	CounterName  string
	CounterValue string
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
{{if .HasCounter}}
<h2>Counters</h2>
<ul><li>{{.CounterName}}: {{.CounterValue}}</li></ul>
{{end}}
{{if and (not .Gauges) (not .HasCounter)}}<p>No metrics yet.</p>{{end}}
</body>
</html>
`))

func NewUpdateMetricsHandler(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveUpdateMetrics(w, r, store)
	}
}

func serveUpdateMetrics(w http.ResponseWriter, r *http.Request, store repository.Storage) {
	mtype := strings.ToLower(chi.URLParam(r, "mtype"))
	name := chi.URLParam(r, "name")
	valueStr := chi.URLParam(r, "value")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	if valueStr == "" {
		http.Error(w, "missing metric value", http.StatusBadRequest)
		return
	}

	switch mtype {
	case models.Gauge:
		v, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
		store.SetGauge(name, v)
		log.Printf("server: gauge %s = %g", name, v)
	case models.Counter:
		v, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
		store.AddCounter(name, v)
		log.Printf("server: counter %s += %d", name, v)
	default:
		if mtype == "" {
			http.Error(w, "missing metric type", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func NewValueHandler(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveMetricValue(w, r, store)
	}
}

func serveMetricValue(w http.ResponseWriter, r *http.Request, store repository.Storage) {
	mtype := strings.ToLower(chi.URLParam(r, "mtype"))
	name := chi.URLParam(r, "name")

	if name == "" {
		http.NotFound(w, r)
		return
	}

	switch mtype {
	case models.Gauge:
		v, ok := store.GetGauge(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatFloat(v, 'g', -1, 64)))
	case models.Counter:
		v, ok := store.GetCounter(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(v, 10)))
	default:
		if mtype == "" {
			http.Error(w, "missing metric type", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
	}
}

func NewMetricHandler(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveMetricsValue(w, store)
	}
}

func serveMetricsValue(w http.ResponseWriter, store repository.Storage) {
	gauges := store.Gauges()

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

	cVal, cOk := store.GetCounter(models.PollCountName)

	var buf bytes.Buffer
	data := metricsIndexData{
		Gauges:       gRows,
		HasCounter:   cOk,
		CounterName:  models.PollCountName,
		CounterValue: strconv.FormatInt(cVal, 10),
	}
	if err := metricsIndexTmpl.Execute(&buf, data); err != nil {
		log.Printf("metrics index template: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
