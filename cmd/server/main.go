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
	"github.com/Radiushina/metrics-receiver.git/internal/middleware"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	if exitCode, err := parseFlags(); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(os.Stderr, err)
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

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repos := repository.NewRepository()
	svc := service.NewService(repos)

	fileStorage := repository.NewFileStorage(repos, flagFileStoragePath)
	if flagRestore {
		if err := fileStorage.Restore(ctx); err != nil {
			return err
		}
	}

	var saver handler.Saver
	if flagStoreIntervalSec == 0 {
		saver = fileStorage
	}

	h := handler.NewHandler(svc, saver)

	logger.Log.Info("starting metrics server on",
		zap.String("address", flagRunAddr))
	srv := &Server{}
	mux := NewMux(h)

	if flagStoreIntervalSec > 0 {
		interval := time.Duration(flagStoreIntervalSec) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ticker.C:
					if err := fileStorage.Save(ctx); err != nil {
						logger.Log.Warn("failed to persist metrics",
							zap.Error(err))
					}
				case <-ctx.Done():
					_ = fileStorage.Save(context.Background())
					return
				}
			}
		}()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(mux)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Print("shutting down metrics server")
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
	r.Use(middleware.DecompressRequest)
	r.Use(logger.LoggingMiddleware)
	r.Use(middleware.CompressResponse)
	r.Get("/", h.GetMetrics())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Post("/update", h.UpdateFromBody())
	r.Post("/update/", h.UpdateFromBody())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	r.Post("/value", h.GetMetricValue())
	r.Post("/value/", h.GetMetricValue())
	return r
}
