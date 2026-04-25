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

const (
	defaultRunAddr          = ":8080"
	defaultLogLevel         = "info"
	defaultStoreIntervalSec = 300
	defaultFileStoragePath  = "./metrics-db.json"
)

var (
	flagRunAddr          string
	flagLogLevel         string
	flagStoreIntervalSec int
	flagFileStoragePath  string
	flagRestore          bool
)

func parseFlags() (exitCode int, err error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registerFlags(fs)
	if exitCode, err := parseCLI(fs); err != nil {
		return exitCode, err
	}

	var envCfg config.ServiceConfig
	if err := env.Parse(&envCfg); err != nil {
		return 1, err
	}

	applyEnvConfig(envCfg)
	applyEnvLogLevel()
	normalize()
	if err := validate(); err != nil {
		return 1, err
	}

	return 0, nil
}

func registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&flagRunAddr, "a", defaultRunAddr, "address and port to run server")
	fs.StringVar(&flagLogLevel, "l", defaultLogLevel, "log level")
	fs.IntVar(&flagStoreIntervalSec, "i", defaultStoreIntervalSec, "store interval in seconds (0 means synchronous)")
	fs.StringVar(&flagFileStoragePath, "f", defaultFileStoragePath, "file path to persist metrics")
	fs.BoolVar(&flagRestore, "r", false, "restore persisted metrics on startup")
}

func parseCLI(fs *flag.FlagSet) (exitCode int, err error) {
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, fmt.Errorf("ошибка флагов: %w", err)
	}
	return 0, nil
}

func applyEnvConfig(envCfg config.ServiceConfig) {
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
}

func applyEnvLogLevel() {
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
}

func normalize() {
	flagRunAddr = strings.TrimSpace(flagRunAddr)
	flagFileStoragePath = strings.TrimSpace(flagFileStoragePath)
}

func validate() error {
	if flagStoreIntervalSec < 0 {
		return fmt.Errorf("STORE_INTERVAL is negative %d", flagStoreIntervalSec)
	}
	if flagFileStoragePath == "" {
		return errors.New("FILE_STORAGE_PATH is empty")
	}
	return nil
}
