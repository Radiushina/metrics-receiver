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
)

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
	cryptoKey            string
	flagConfigPath       string
)

// parseFlags загружает конфиг в порядке: defaults → JSON → flags → ENV.
func parseFlags() (exitCode int, err error) {
	flagRunAddr = defaultRunAddr
	flagLogLevel = defaultLogLevel
	flagStoreIntervalSec = defaultStoreIntervalSec
	flagFileStoragePath = defaultFileStoragePath
	flagDatabaseDSN = defaultDatabaseDSN
	flagSecretKey = defaultSecretKey
	flagRestore = false
	flagAuditFilePath = ""
	flagAuditURL = ""
	cryptoKey = ""
	flagConfigPath = ""

	configPath := config.ResolveConfigPath(os.Args[1:])
	if configPath != "" {
		fileCfg, err := config.LoadServerFile(configPath)
		if err != nil {
			return 1, err
		}
		if err := applyFileConfig(fileCfg); err != nil {
			return 1, err
		}
		flagConfigPath = configPath
	}

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
	fs.StringVar(&flagConfigPath, "c", flagConfigPath, "path to JSON config file")
	fs.StringVar(&flagConfigPath, "config", flagConfigPath, "path to JSON config file")
	fs.StringVar(&flagRunAddr, "a", flagRunAddr, "address and port to run server")
	fs.StringVar(&flagLogLevel, "l", flagLogLevel, "log level")
	fs.IntVar(&flagStoreIntervalSec, "i", flagStoreIntervalSec, "store interval in seconds (0 means synchronous)")
	fs.StringVar(&flagFileStoragePath, "f", flagFileStoragePath, "file path to persist metrics")
	fs.BoolVar(&flagRestore, "r", flagRestore, "restore persisted metrics on startup")
	fs.StringVar(&flagDatabaseDSN, "d", flagDatabaseDSN, "database dsn")
	fs.StringVar(&flagSecretKey, "k", flagSecretKey, "shared secret for HMAC-SHA256 (HashSHA256 header); empty disables signing")
	fs.StringVar(&flagAuditFilePath, "audit-file", flagAuditFilePath, "path to audit file")
	fs.StringVar(&flagAuditURL, "audit-url", flagAuditURL, "url to send audit logs")
	fs.StringVar(&cryptoKey, "crypto-key", cryptoKey, "path to private key")
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

func applyFileConfig(cfg config.ServerFileConfig) error {
	if cfg.Address != nil {
		flagRunAddr = strings.TrimSpace(*cfg.Address)
	}
	if cfg.Restore != nil {
		flagRestore = *cfg.Restore
	}
	if cfg.StoreInterval != nil {
		sec, err := config.DurationSeconds(*cfg.StoreInterval)
		if err != nil {
			return fmt.Errorf("store_interval: %w", err)
		}
		flagStoreIntervalSec = int(sec)
	}
	if cfg.StoreFile != nil {
		flagFileStoragePath = strings.TrimSpace(*cfg.StoreFile)
	}
	if cfg.DatabaseDSN != nil {
		flagDatabaseDSN = strings.TrimSpace(*cfg.DatabaseDSN)
	}
	if cfg.CryptoKey != nil {
		cryptoKey = strings.TrimSpace(*cfg.CryptoKey)
	}
	if cfg.Key != nil {
		flagSecretKey = strings.TrimSpace(*cfg.Key)
	}
	if cfg.AuditFile != nil {
		flagAuditFilePath = strings.TrimSpace(*cfg.AuditFile)
	}
	if cfg.AuditURL != nil {
		flagAuditURL = strings.TrimSpace(*cfg.AuditURL)
	}
	if cfg.LogLevel != nil {
		flagLogLevel = strings.TrimSpace(*cfg.LogLevel)
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
		cryptoKey = strings.TrimSpace(*envCfg.CryptoKey)
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
	cryptoKey = strings.TrimSpace(cryptoKey)
	flagConfigPath = strings.TrimSpace(flagConfigPath)
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
