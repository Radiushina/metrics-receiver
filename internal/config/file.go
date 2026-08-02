package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AgentFileConfig — JSON-конфиг агента (-c / CONFIG).
type AgentFileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"` // например "1s"
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	Key            *string `json:"key"`
	RateLimit      *int64  `json:"rate_limit"`
}

// ServerFileConfig — JSON-конфиг сервера (-c / CONFIG).
type ServerFileConfig struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"` // например "1s"
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	Key           *string `json:"key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
	LogLevel      *string `json:"log_level"`
}

// ResolveConfigPath возвращает путь к JSON-конфигу.
// Приоритет: флаг -c/-config выше, чем CONFIG из окружения.
func ResolveConfigPath(args []string) string {
	path := strings.TrimSpace(os.Getenv("CONFIG"))
	if fromArgs := configPathFromArgs(args); fromArgs != "" {
		path = fromArgs
	}
	return strings.TrimSpace(path)
}

func configPathFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "-config" || a == "--config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		case strings.HasPrefix(a, "-config="):
			return strings.TrimPrefix(a, "-config=")
		case strings.HasPrefix(a, "--config="):
			return strings.TrimPrefix(a, "--config=")
		}
	}
	return ""
}

// LoadAgentFile читает JSON-конфиг агента.
func LoadAgentFile(path string) (AgentFileConfig, error) {
	var cfg AgentFileConfig
	if err := loadJSON(path, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadServerFile читает JSON-конфиг сервера.
func LoadServerFile(path string) (ServerFileConfig, error) {
	var cfg ServerFileConfig
	if err := loadJSON(path, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func loadJSON(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config file %q: %w", path, err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("config file %q: %w", path, err)
	}
	return nil
}

// DurationSeconds парсит значение интервала из JSON ("1s", "300s") или целое число секунд.
func DurationSeconds(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return int64(d / time.Second), nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	return n, nil
}
