package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/config"
	"github.com/caarlos0/env/v11"
)

var (
	flagRunAddr          string
	flagLogLevel         string
	flagStoreIntervalSec int
	flagFileStoragePath  string
	flagRestore          bool
)

func parseFlags() (exitCode int, err error) {
	var envCfg config.ServiceConfig

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	fs.StringVar(&flagLogLevel, "l", "info", "log level")
	fs.IntVar(&flagStoreIntervalSec, "i", 300, "store interval in seconds (0 means synchronous)")
	fs.StringVar(&flagFileStoragePath, "f", "./metrics-db.json", "file path to persist metrics")
	fs.BoolVar(&flagRestore, "r", false, "restore persisted metrics on startup")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, fmt.Errorf("ошибка флагов: %v\n", err)
	}

	if err := env.Parse(&envCfg); err != nil {
		return 1, err
	}

	if envCfg.RunAddr != nil {
		flagRunAddr = strings.TrimSpace(*envCfg.RunAddr)
	}
	if envCfg.StoreInterval != nil {
		flagStoreIntervalSec = *envCfg.StoreInterval
	}
	if envCfg.FileStoragePath != nil {
		flagFileStoragePath = strings.TrimSpace(*envCfg.FileStoragePath)
	}
	if envCfg.Restore != nil {
		flagRestore = *envCfg.Restore
	}

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}

	flagRunAddr = strings.TrimSpace(flagRunAddr)
	flagFileStoragePath = strings.TrimSpace(flagFileStoragePath)
	if flagStoreIntervalSec < 0 {
		return 1, fmt.Errorf("STORE_INTERVAL must be non-negative, got %d", flagStoreIntervalSec)
	}
	if flagFileStoragePath == "" {
		return 1, fmt.Errorf("FILE_STORAGE_PATH must be non-empty")
	}

	return 0, nil
}
