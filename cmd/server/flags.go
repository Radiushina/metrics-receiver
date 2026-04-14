package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Radiushina/metrics-receiver.git/internal/config"
	"github.com/caarlos0/env/v6"
)

var (
	flagRunAddr  string
	flagLogLevel string
)

func parseFlags() (exitCode int, err error) {
	var envCfg config.ServiceConfig

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	fs.StringVar(&flagLogLevel, "l", "info", "log level")

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

	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}

	return 0, nil
}
