package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPath_FlagBeatsEnv(t *testing.T) {
	t.Setenv("CONFIG", "/from/env.json")
	visited := map[string]bool{"c": true}
	got := ResolveConfigPath("/from/flag.json", visited)
	if got != "/from/flag.json" {
		t.Fatalf("got %q, want flag path", got)
	}
}

func TestResolveConfigPath_EnvOnly(t *testing.T) {
	t.Setenv("CONFIG", "/from/env.json")
	got := ResolveConfigPath("", map[string]bool{})
	if got != "/from/env.json" {
		t.Fatalf("got %q, want env path", got)
	}
}

func TestVisitedFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var a, c string
	fs.StringVar(&a, "a", "default", "")
	fs.StringVar(&c, "c", "", "")
	if err := fs.Parse([]string{"-c", "cfg.json"}); err != nil {
		t.Fatal(err)
	}
	visited := VisitedFlags(fs)
	if !visited["c"] {
		t.Fatal("expected c visited")
	}
	if visited["a"] {
		t.Fatal("a must not be visited")
	}
}

func TestDurationSeconds(t *testing.T) {
	sec, err := DurationSeconds("1s")
	if err != nil || sec != 1 {
		t.Fatalf("1s: got %d %v", sec, err)
	}
	sec, err = DurationSeconds("300s")
	if err != nil || sec != 300 {
		t.Fatalf("300s: got %d %v", sec, err)
	}
	sec, err = DurationSeconds("10")
	if err != nil || sec != 10 {
		t.Fatalf("10: got %d %v", sec, err)
	}
}

func TestLoadAgentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	content := `{
  "address": "localhost:9090",
  "report_interval": "5s",
  "poll_interval": "2s",
  "crypto_key": "/pub.pem",
  "key": "secret",
  "rate_limit": 3
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadAgentFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Address == nil || *cfg.Address != "localhost:9090" {
		t.Fatalf("address: %+v", cfg.Address)
	}
	sec, err := DurationSeconds(*cfg.ReportInterval)
	if err != nil || sec != 5 {
		t.Fatalf("report_interval: %v %v", sec, err)
	}
	if cfg.RateLimit == nil || *cfg.RateLimit != 3 {
		t.Fatalf("rate_limit: %+v", cfg.RateLimit)
	}
}

func TestLoadServerFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.json")
	content := `{
  "address": ":9090",
  "restore": true,
  "store_interval": "1s",
  "store_file": "/tmp/db.json",
  "database_dsn": "postgres://x",
  "crypto_key": "/priv.pem"
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadServerFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Restore == nil || !*cfg.Restore {
		t.Fatal("restore")
	}
	sec, err := DurationSeconds(*cfg.StoreInterval)
	if err != nil || sec != 1 {
		t.Fatalf("store_interval: %v %v", sec, err)
	}
	if cfg.StoreFile == nil || *cfg.StoreFile != "/tmp/db.json" {
		t.Fatalf("store_file: %+v", cfg.StoreFile)
	}
}
