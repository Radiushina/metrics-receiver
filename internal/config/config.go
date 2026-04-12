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
type ServiceConfig struct {
	RunAddr *string `env:"ADDRESS"`
}
