package config

// AgentConfig holds optional environment overrides for the metrics agent.
// Pointer fields are nil when the corresponding variable is not set in the environment.
// Interval env vars are interpreted as whole seconds (integers).
type AgentConfig struct {
	RunAddr           *string `env:"ADDRESS"`
	PollIntervalSec   *int64  `env:"POLL_INTERVAL"`
	ReportIntervalSec *int64  `env:"REPORT_INTERVAL"`
}

// ServiceConfig holds optional environment overrides for the metrics server.
// Pointer fields are nil when the corresponding variable is not set in the environment.
//
// STORE_INTERVAL is interpreted as whole seconds. A value of 0 enables synchronous persistence
// (save after each update); a positive value enables periodic persistence with the given interval.
// FILE_STORAGE_PATH is the path to the file used for persistence.
// RESTORE controls whether persisted metrics should be loaded from FILE_STORAGE_PATH on startup.
type ServiceConfig struct {
	RunAddr         *string `env:"ADDRESS"`
	StoreInterval   *int    `env:"STORE_INTERVAL"`
	FileStoragePath *string `env:"FILE_STORAGE_PATH"`
	Restore         *bool   `env:"RESTORE"`
}
