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

const (
	defaultAgentRunAddr        = "localhost:8080"
	defaultAgentPollInterval   = int64(2)
	defaultAgentReportInterval = int64(10)
	defaultAgentKey            = ""
	defaultAgentRateLimit      = int64(1)
	defaultAgentCryptoKey      = ""
	defaultAgentConfigPath     = ""
	defaultAgentGRPCAddr       = ""
)

type Flags struct {
	runAddr        string
	pollInterval   int64
	reportInterval int64
	key            string
	rateLimit      int64
	cryptoKey      string
	configPath     string
	grpcAddr       string
}

func NewFlags() *Flags {
	return &Flags{
		runAddr:        defaultAgentRunAddr,
		pollInterval:   defaultAgentPollInterval,
		reportInterval: defaultAgentReportInterval,
		key:            defaultAgentKey,
		rateLimit:      defaultAgentRateLimit,
		cryptoKey:      defaultAgentCryptoKey,
		configPath:     defaultAgentConfigPath,
		grpcAddr:       defaultAgentGRPCAddr,
	}
}

// parse загружает конфиг в порядке: defaults → flags → JSON (только незаданные) → ENV.
func (r *Flags) parse() (exitCode int, err error) {
	*r = *NewFlags()

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&r.configPath, "c", defaultAgentConfigPath, "path to JSON config file")
	fs.StringVar(&r.configPath, "config", defaultAgentConfigPath, "path to JSON config file")
	fs.StringVar(&r.runAddr, "a", defaultAgentRunAddr, "HTTP server host:port (scheme http:// added automatically)")
	fs.Int64Var(&r.pollInterval, "p", defaultAgentPollInterval, "poll interval: how often to read runtime.MemStats (seconds)")
	fs.Int64Var(&r.reportInterval, "r", defaultAgentReportInterval, "report interval: how often to send metrics to the server (seconds)")
	fs.StringVar(&r.key, "k", defaultAgentKey, "shared secret for HMAC-SHA256 request body signature (HashSHA256 header); empty disables signing")
	fs.Int64Var(&r.rateLimit, "l", defaultAgentRateLimit, "max number of concurrent outgoing HTTP requests to the server")
	fs.StringVar(&r.cryptoKey, "crypto-key", defaultAgentCryptoKey, "path to public key")
	fs.StringVar(&r.grpcAddr, "g", defaultAgentGRPCAddr, "gRPC server address (host:port); if set, metrics are sent via gRPC")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, flag.ErrHelp
		}
		return 1, err
	}

	visited := config.VisitedFlags(fs)
	configPath := config.ResolveConfigPath(r.configPath, visited)
	if configPath != "" {
		fileCfg, err := config.LoadAgentFile(configPath)
		if err != nil {
			return 1, err
		}
		if err := r.applyFileIfUnset(fileCfg, visited); err != nil {
			return 1, err
		}
		if !visited["c"] && !visited["config"] {
			r.configPath = configPath
		}
	}

	var envCfg config.AgentConfig
	if err := env.Parse(&envCfg); err != nil {
		return 1, err
	}
	r.applyEnv(envCfg)

	return 0, nil
}

func (r *Flags) applyFileIfUnset(cfg config.AgentFileConfig, visited map[string]bool) error {
	if cfg.Address != nil && !visited["a"] {
		r.runAddr = strings.TrimSpace(*cfg.Address)
	}
	if cfg.PollInterval != nil && !visited["p"] {
		sec, err := config.DurationSeconds(*cfg.PollInterval)
		if err != nil {
			return fmt.Errorf("poll_interval: %w", err)
		}
		r.pollInterval = sec
	}
	if cfg.ReportInterval != nil && !visited["r"] {
		sec, err := config.DurationSeconds(*cfg.ReportInterval)
		if err != nil {
			return fmt.Errorf("report_interval: %w", err)
		}
		r.reportInterval = sec
	}
	if cfg.CryptoKey != nil && !visited["crypto-key"] {
		r.cryptoKey = strings.TrimSpace(*cfg.CryptoKey)
	}
	if cfg.Key != nil && !visited["k"] {
		r.key = strings.TrimSpace(*cfg.Key)
	}
	if cfg.RateLimit != nil && !visited["l"] {
		r.rateLimit = *cfg.RateLimit
	}
	if cfg.GRPCAddress != nil && !visited["g"] {
		r.grpcAddr = strings.TrimSpace(*cfg.GRPCAddress)
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
	if envCfg.GRPCAddr != nil {
		r.grpcAddr = strings.TrimSpace(*envCfg.GRPCAddr)
	}
}

func (r *Flags) serverBaseURL() string {
	s := strings.TrimSpace(r.runAddr)
	if s == "" {
		s = defaultAgentRunAddr
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
