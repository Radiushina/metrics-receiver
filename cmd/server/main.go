package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	if exitCode, err := parseFlags(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}

	if err := run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func run() error {
	repos := repository.NewRepository()
	svc := service.NewService(repos)
	h := handler.NewHandler(svc)

	log.Printf("starting metrics server on %s", flagRunAddr)
	return http.ListenAndServe(flagRunAddr, NewMux(h))
}

func NewMux(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.GetMetrics())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	r.Post("/update/{mtype}/{metric}/{value}", h.Update())
	return r
}
