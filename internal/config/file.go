package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"errors"
)

// AgentFileConfig — JSON-конфиг агента (-c / CONFIG).
type AgentFileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"` // например "1s"
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	Key            *string `json:"key"`
	RateLimit      *int64  `json:"rate_limit"`
	GRPCAddress    *string `json:"grpc_address"`
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
	TrustedSubnet *string `json:"trusted_subnet"`
	GRPCAddress   *string `json:"grpc_address"`
}

// VisitedFlags возвращает имена флагов, явно переданных в argv (fs.Visit).
func VisitedFlags(fs *flag.FlagSet) map[string]bool {
	out := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

// ResolveConfigPath выбирает путь к JSON-конфигу.
// Если в argv был -c/-config — берётся flagPath, иначе CONFIG из окружения.
func ResolveConfigPath(flagPath string, visited map[string]bool) string {
	if visited["c"] || visited["config"] {
		return strings.TrimSpace(flagPath)
	}
	return strings.TrimSpace(os.Getenv("CONFIG"))
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
		return 0, errors.New("empty duration")
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
