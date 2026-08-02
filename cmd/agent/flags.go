package main

import (
	"errors"
	"flag"
	"fmt"
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
	rateLimit      int64
	cryptoKey      string
	configPath     string
}

func NewFlags() *Flags {
	return &Flags{
		runAddr:        "localhost:8080",
		pollInterval:   2,
		reportInterval: 10,
		key:            "",
		rateLimit:      1,
		cryptoKey:      "",
	}
}

// parse загружает конфиг в порядке: defaults → JSON → flags → ENV.
func (r *Flags) parse() (exitCode int, err error) {
	configPath := config.ResolveConfigPath(os.Args[1:])
	if configPath != "" {
		fileCfg, err := config.LoadAgentFile(configPath)
		if err != nil {
			return 1, err
		}
		if err := r.applyFile(fileCfg); err != nil {
			return 1, err
		}
		r.configPath = configPath
	}

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&r.configPath, "c", r.configPath, "path to JSON config file")
	fs.StringVar(&r.configPath, "config", r.configPath, "path to JSON config file")
	fs.StringVar(&r.runAddr, "a", r.runAddr, "HTTP server host:port (scheme http:// added automatically)")
	fs.Int64Var(&r.pollInterval, "p", r.pollInterval, "poll interval: how often to read runtime.MemStats (seconds)")
	fs.Int64Var(&r.reportInterval, "r", r.reportInterval, "report interval: how often to send metrics to the server (seconds)")
	fs.StringVar(&r.key, "k", r.key, "shared secret for HMAC-SHA256 request body signature (HashSHA256 header); empty disables signing")
	fs.Int64Var(&r.rateLimit, "l", r.rateLimit, "max number of concurrent outgoing HTTP requests to the server")
	fs.StringVar(&r.cryptoKey, "crypto-key", r.cryptoKey, "path to public key")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, err
	}

	var envCfg config.AgentConfig
	if err := env.Parse(&envCfg); err != nil {
		return 1, err
	}
	r.applyEnv(envCfg)

	return 0, nil
}

func (r *Flags) applyFile(cfg config.AgentFileConfig) error {
	if cfg.Address != nil {
		r.runAddr = strings.TrimSpace(*cfg.Address)
	}
	if cfg.PollInterval != nil {
		sec, err := config.DurationSeconds(*cfg.PollInterval)
		if err != nil {
			return fmt.Errorf("poll_interval: %w", err)
		}
		r.pollInterval = sec
	}
	if cfg.ReportInterval != nil {
		sec, err := config.DurationSeconds(*cfg.ReportInterval)
		if err != nil {
			return fmt.Errorf("report_interval: %w", err)
		}
		r.reportInterval = sec
	}
	if cfg.CryptoKey != nil {
		r.cryptoKey = strings.TrimSpace(*cfg.CryptoKey)
	}
	if cfg.Key != nil {
		r.key = strings.TrimSpace(*cfg.Key)
	}
	if cfg.RateLimit != nil {
		r.rateLimit = *cfg.RateLimit
	}
	return nil
}

func (r *Flags) applyEnv(envCfg config.AgentConfig) {
	if envCfg.RunAddr != nil {
		r.runAddr = strings.TrimSpace(*envCfg.RunAddr)
	}
	if envCfg.PollIntervalSec != nil {
		r.pollInterval = *envCfg.PollIntervalSec
	}
	if envCfg.ReportIntervalSec != nil {
		r.reportInterval = *envCfg.ReportIntervalSec
	}
	if envCfg.Key != nil {
		r.key = strings.TrimSpace(*envCfg.Key)
	}
	if envCfg.RateLimit != nil {
		r.rateLimit = *envCfg.RateLimit
	}
	if envCfg.CryptoKey != nil {
		r.cryptoKey = strings.TrimSpace(*envCfg.CryptoKey)
	}
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
