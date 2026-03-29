package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

func NewUpdateMetricsHandler(store repository.Storage) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveUpdateMetrics(w, r, store)
	})
}

func serveUpdateMetrics(w http.ResponseWriter, r *http.Request, store repository.Storage) {
	mtype := strings.ToLower(r.PathValue("mtype"))
	name := r.PathValue("name")
	valueStr := r.PathValue("value")

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
