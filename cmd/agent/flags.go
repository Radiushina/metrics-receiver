package main

import (
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/config"
	"github.com/caarlos0/env/v11"
)

var flagRunAddr string
var flagPollInterval int64
var flagReportInterval int64

func parseFlags() (exitCode int, err error) {
	var envCfg config.AgentConfig

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&flagRunAddr, "a", "localhost:8080", "HTTP server host:port (scheme http:// added automatically)")
	fs.Int64Var(&flagPollInterval, "p", 2, "poll interval: how often to read runtime.MemStats (seconds)")
	fs.Int64Var(&flagReportInterval, "r", 10, "report interval: how often to send metrics to the server (seconds)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, err
	}

	if err := env.Parse(&envCfg); err != nil {
		return 1, err
	}

	if envCfg.RunAddr != nil {
		flagRunAddr = strings.TrimSpace(*envCfg.RunAddr)
	}
	if envCfg.PollIntervalSec != nil {
		flagPollInterval = *envCfg.PollIntervalSec
	}
	if envCfg.ReportIntervalSec != nil {
		flagReportInterval = *envCfg.ReportIntervalSec
	}

	return 0, nil
}

func serverBaseURL() string {
	s := strings.TrimSpace(flagRunAddr)
	if s == "" {
		s = "localhost:8080"
	}
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "/")
	return "http://" + s
}

func getPollInterval() time.Duration {
	return time.Duration(flagPollInterval) * time.Second
}

func getReportInterval() time.Duration {
	return time.Duration(flagReportInterval) * time.Second
}
