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
	defaultDatabaseDSN      = ""
	defaultSecretKey        = ""
	defaultRestore          = false
	defaultAuditFilePath    = ""
	defaultAuditURL         = ""
	defaultCryptoKey        = ""
	defaultConfigPath       = ""
	defaultTrustedSubnet    = ""
)

// Итоговые значения после parseFlags (defaults → flags → JSON для незаданных → ENV).
var (
	flagRunAddr          string
	flagLogLevel         string
	flagStoreIntervalSec int
	flagFileStoragePath  string
	flagRestore          bool
	flagDatabaseDSN      string
	flagSecretKey        string
	flagAuditFilePath    string
	flagAuditURL         string
	flagCryptoKey        string
	flagConfigPath       string
	flagTrustedSubnet    string
)

// parseFlags загружает конфиг в порядке: defaults → flags → JSON (только незаданные флаги) → ENV.
func parseFlags() (exitCode int, err error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	registerFlags(fs)

	if exitCode, err := parseCLI(fs); err != nil {
		return exitCode, err
	}

	visited := config.VisitedFlags(fs)
	configPath := config.ResolveConfigPath(flagConfigPath, visited)
	if configPath != "" {
		fileCfg, err := config.LoadServerFile(configPath)
		if err != nil {
			return 1, err
		}
		if err := applyFileConfigIfUnset(fileCfg, visited); err != nil {
			return 1, err
		}
		if !visited["c"] && !visited["config"] {
			flagConfigPath = configPath
		}
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
	fs.StringVar(&flagConfigPath, "c", defaultConfigPath, "path to JSON config file")
	fs.StringVar(&flagConfigPath, "config", defaultConfigPath, "path to JSON config file")
	fs.StringVar(&flagRunAddr, "a", defaultRunAddr, "address and port to run server")
	fs.StringVar(&flagLogLevel, "l", defaultLogLevel, "log level")
	fs.IntVar(&flagStoreIntervalSec, "i", defaultStoreIntervalSec, "store interval in seconds (0 means synchronous)")
	fs.StringVar(&flagFileStoragePath, "f", defaultFileStoragePath, "file path to persist metrics")
	fs.BoolVar(&flagRestore, "r", defaultRestore, "restore persisted metrics on startup")
	fs.StringVar(&flagDatabaseDSN, "d", defaultDatabaseDSN, "database dsn")
	fs.StringVar(&flagSecretKey, "k", defaultSecretKey, "shared secret for HMAC-SHA256 (HashSHA256 header); empty disables signing")
	fs.StringVar(&flagAuditFilePath, "audit-file", defaultAuditFilePath, "path to audit file")
	fs.StringVar(&flagAuditURL, "audit-url", defaultAuditURL, "url to send audit logs")
	fs.StringVar(&flagCryptoKey, "crypto-key", defaultCryptoKey, "path to private key")
	fs.StringVar(&flagTrustedSubnet, "t", defaultTrustedSubnet, "trusted subnet")
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

// applyFileConfigIfUnset применяет JSON только к опциям, не заданным явно во флагах.
func applyFileConfigIfUnset(cfg config.ServerFileConfig, visited map[string]bool) error {
	if cfg.Address != nil && !visited["a"] {
		flagRunAddr = strings.TrimSpace(*cfg.Address)
	}
	if cfg.Restore != nil && !visited["r"] {
		flagRestore = *cfg.Restore
	}
	if cfg.StoreInterval != nil && !visited["i"] {
		sec, err := config.DurationSeconds(*cfg.StoreInterval)
		if err != nil {
			return fmt.Errorf("store_interval: %w", err)
		}
		flagStoreIntervalSec = int(sec)
	}
	if cfg.StoreFile != nil && !visited["f"] {
		flagFileStoragePath = strings.TrimSpace(*cfg.StoreFile)
	}
	if cfg.DatabaseDSN != nil && !visited["d"] {
		flagDatabaseDSN = strings.TrimSpace(*cfg.DatabaseDSN)
	}
	if cfg.CryptoKey != nil && !visited["crypto-key"] {
		flagCryptoKey = strings.TrimSpace(*cfg.CryptoKey)
	}
	if cfg.Key != nil && !visited["k"] {
		flagSecretKey = strings.TrimSpace(*cfg.Key)
	}
	if cfg.AuditFile != nil && !visited["audit-file"] {
		flagAuditFilePath = strings.TrimSpace(*cfg.AuditFile)
	}
	if cfg.AuditURL != nil && !visited["audit-url"] {
		flagAuditURL = strings.TrimSpace(*cfg.AuditURL)
	}
	if cfg.LogLevel != nil && !visited["l"] {
		flagLogLevel = strings.TrimSpace(*cfg.LogLevel)
	}
	if cfg.TrustedSubnet != nil && !visited["t"] {
		flagTrustedSubnet = strings.TrimSpace(*cfg.TrustedSubnet)
	}
	return nil
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
	if envCfg.DbDsn != nil {
		flagDatabaseDSN = *envCfg.DbDsn
	}
	if envCfg.Key != nil {
		flagSecretKey = *envCfg.Key
	}
	if envCfg.AuditFilePath != nil {
		flagAuditFilePath = strings.TrimSpace(*envCfg.AuditFilePath)
	}
	if envCfg.AuditURL != nil {
		flagAuditURL = strings.TrimSpace(*envCfg.AuditURL)
	}
	if envCfg.CryptoKey != nil {
		flagCryptoKey = strings.TrimSpace(*envCfg.CryptoKey)
	}
	if envCfg.TrustedSubnet != nil {
		flagTrustedSubnet = strings.TrimSpace(*envCfg.TrustedSubnet)
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
	flagDatabaseDSN = strings.TrimSpace(flagDatabaseDSN)
	flagSecretKey = strings.TrimSpace(flagSecretKey)
	flagAuditFilePath = strings.TrimSpace(flagAuditFilePath)
	flagAuditURL = strings.TrimSpace(flagAuditURL)
	flagCryptoKey = strings.TrimSpace(flagCryptoKey)
	flagConfigPath = strings.TrimSpace(flagConfigPath)
	flagTrustedSubnet = strings.TrimSpace(flagTrustedSubnet)
}

func validate() error {
	if flagStoreIntervalSec < 0 {
		return fmt.Errorf("STORE_INTERVAL is negative %d", flagStoreIntervalSec)
	}
	if flagFileStoragePath == "" {
		panic("FILE_STORAGE_PATH is empty")
	}
	return nil
}
