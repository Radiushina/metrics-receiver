package main

import (
	"log"
	"net/http"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/go-chi/chi/v5"
)

func main() {
	parseFlags()

	if err := run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func run() error {
	store := repository.NewMemStorage()
	log.Printf("starting metrics server on %s", flagRunAddr)
	return http.ListenAndServe(flagRunAddr, newMux(store))
}

func newMux(store repository.Storage) http.Handler {
	r := chi.NewRouter()
	r.Get("/", handler.NewMetricHandler(store))
	r.Get("/value/{mtype}/{metric}", handler.NewValueHandler(store))
	r.Post("/update/{mtype}/{metric}/{value}", handler.NewUpdateMetricsHandler(store))
	return r
}
