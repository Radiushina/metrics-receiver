package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	if exitCode, err := parseFlags(); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode)
	}

	if err := run(); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func run() error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	repos := repository.NewRepository()
	svc := service.NewService(repos)
	h := handler.NewHandler(svc)

	logger.Log.Info("starting metrics server on", zap.String("address", flagRunAddr))
	srv := &Server{}
	mux := NewMux(h)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(mux)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("shutting down metrics server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	runErr := <-errCh

	if errors.Is(runErr, http.ErrServerClosed) {
		runErr = nil
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	return runErr
}

func NewMux(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(logger.LoggingMiddleware)
	r.Get("/", h.GetMetrics())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Post("/update", h.UpdateFromBody())
	r.Post("/update/", h.UpdateFromBody())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	r.Post("/value", h.GetMetricValue())
	r.Post("/value/", h.GetMetricValue())
	return r
}
