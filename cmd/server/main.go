package main

import (
	"log"
	"net/http"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
)

func main() {
	if err := run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func run() error {
	store := repository.NewMemStorage()
	mux := http.NewServeMux()
	mux.Handle("POST /update/{mtype}/{name}/{value}", handler.NewUpdateMetricsHandler(store))

	log.Println("starting metrics server on :8080")
	return http.ListenAndServe(":8080", mux)
}
