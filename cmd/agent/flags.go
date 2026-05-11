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

type Flags struct {
	runAddr        string
	pollInterval   int64
	reportInterval int64
	key            string
}

func NewFlags() *Flags {
	return &Flags{
		runAddr:        "localhost:8080",
		pollInterval:   2,
		reportInterval: 10,
		key:            "",
	}
}

func (r *Flags) parse() (exitCode int, err error) {
	var envCfg config.AgentConfig

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&r.runAddr, "a", r.runAddr, "HTTP server host:port (scheme http:// added automatically)")
	fs.Int64Var(&r.pollInterval, "p", r.pollInterval, "poll interval: how often to read runtime.MemStats (seconds)")
	fs.Int64Var(&r.reportInterval, "r", r.reportInterval, "report interval: how often to send metrics to the server (seconds)")
	fs.StringVar(&r.key, "k", r.key, "shared secret for HMAC-SHA256 request body signature (HashSHA256 header); empty disables signing")

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
		r.runAddr = strings.TrimSpace(*envCfg.RunAddr)
	}
	if envCfg.PollIntervalSec != nil {
		r.pollInterval = *envCfg.PollIntervalSec
	}
	if envCfg.ReportIntervalSec != nil {
		r.reportInterval = *envCfg.ReportIntervalSec
	}
	if envCfg.KEY != nil {
		r.key = strings.TrimSpace(*envCfg.KEY)
	}

	return 0, nil
}

func (r *Flags) serverBaseURL() string {
	s := strings.TrimSpace(r.runAddr)
	if s == "" {
		s = "localhost:8080"
	}
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "/")
	return "http://" + s
}

func (r *Flags) pollEvery() time.Duration {
	return time.Duration(r.pollInterval) * time.Second
}

func (r *Flags) reportEvery() time.Duration {
	return time.Duration(r.reportInterval) * time.Second
}
