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
	"strings"
	"syscall"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/audit"
	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"github.com/Radiushina/metrics-receiver.git/internal/middleware"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	runServer()
}

func runServer() {
	printBuildInfo()
	if exitCode, err := parseFlags(); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			if _, printErr := fmt.Fprintln(os.Stderr, err); printErr != nil {
				os.Exit(1)
			}
		}
		os.Exit(exitCode)
	}
	if err := run(); err != nil {
		log.Print("Server failed:", err)
		os.Exit(1)
	}
}

func run() error {
	logg, err := logger.New(flagLogLevel)
	if err != nil {
		return err
	}
	logg = logger.OrNop(logg)
	defer func() { _ = logg.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var dbPool *pgxpool.Pool
	defer func() {
		if dbPool != nil {
			dbPool.Close()
		}
	}()

	dsn := strings.TrimSpace(flagDatabaseDSN)
	useDB := dsn != ""

	if useDB {
		p, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("postgres pool: %w", err)
		}
		if err := p.Ping(ctx); err != nil {
			p.Close()
			return fmt.Errorf("postgres ping: %w", err)
		}
		if err := repository.MigrateUp(dsn); err != nil {
			p.Close()
			return err
		}
		dbPool = p
		logg.Info("connected to postgresql", zap.String("dsn", dsn))
	}

	var store service.RepositoryProvider
	var mem *repository.MemoryRepo
	var fileStorage *repository.FileStorage

	if useDB {
		store = repository.NewPostgresRepo(dbPool)
	} else {
		mem = repository.NewMemoryRepo()
		store = mem
		fileStorage = repository.NewFileStorage(mem, flagFileStoragePath)
		if flagRestore {
			if err := fileStorage.Restore(ctx); err != nil {
				return err
			}
		}
	}

	svc := service.NewService(store)

	var saver handler.Saver
	if !useDB && flagStoreIntervalSec == 0 && fileStorage != nil {
		saver = fileStorage
	}

	var dbForPing handler.DBChecker
	if dbPool != nil {
		dbForPing = dbPool
	}

	var fileObs *audit.FileObserver
	auditPub := audit.NewPublisher(ctx, logg)
	if flagAuditFilePath != "" {
		var err error
		fileObs, err = audit.NewFileObserver(flagAuditFilePath, logg)
		if err != nil {
			return err
		}
		auditPub.Register(fileObs)
	}
	defer func() {
		auditPub.Close()
		if fileObs != nil {
			if err := fileObs.Close(); err != nil {
				return
			}
		}
	}()

	if flagAuditURL != "" {
		auditPub.Register(audit.NewHTTPObserver(flagAuditURL, logg))
	}

	h := handler.NewHandler(svc, saver, logg, dbForPing, flagSecretKey, auditPub)

	logg.Info("starting metrics server on",
		zap.String("address", flagRunAddr))
	srv := &Server{}
	mux := NewMux(logg, h)

	if !useDB && flagStoreIntervalSec > 0 && fileStorage != nil {
		interval := time.Duration(flagStoreIntervalSec) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		go func() {
			for {
				select {
				case <-ticker.C:
					if err := fileStorage.Save(ctx); err != nil {
						logg.Warn("failed to persist metrics",
							zap.Error(err))
					}
				case <-ctx.Done():
					if err := fileStorage.Save(context.Background()); err != nil {
						logg.Warn("failed to persist metrics on shutdown", zap.Error(err))
					}
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
		logg.Info("shutting down metrics server")
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

// NewMux собирает chi-роутер с middleware и регистрирует маршруты метрик.
func NewMux(logg *zap.Logger, h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover(logg))
	r.Use(middleware.DecompressRequest)
	r.Use(func(next http.Handler) http.Handler {
		return logger.LoggingMiddleware(logg, next)
	})
	r.Use(middleware.CompressResponse)

	registerRoutes(r, h)
	return r
}

func registerRoutes(r chi.Router, h *handler.Handler) {
	r.Get("/", h.GetMetrics())
	r.Get("/ping", h.PingDB())
	r.Post("/update/{mtype}/{metric}/{value}", h.UpdateFromPath())
	r.Post("/update", h.UpdateFromBody())
	r.Post("/update/", h.UpdateFromBody())
	r.Post("/updates/", h.UpdateMetrics())
	r.Get("/value/{mtype}/{metric}", h.GetMetric())
	r.Post("/value", h.GetMetricValue())
	r.Post("/value/", h.GetMetricValue())
}
