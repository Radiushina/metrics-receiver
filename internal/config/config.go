package config

// AgentConfig holds optional environment overrides for the metrics agent.
// Pointer fields are nil when the corresponding variable is not set in the environment.
// Interval env vars are interpreted as whole seconds (integers).
type AgentConfig struct {
	RunAddr           *string `env:"ADDRESS"`
	PollIntervalSec   *int64  `env:"POLL_INTERVAL"`
	ReportIntervalSec *int64  `env:"REPORT_INTERVAL"`
}

// ServiceConfig содержит опциональные значения из переменных окружения для конфигурации сервера.
// Поля-указатели равны nil, если соответствующая переменная окружения не задана.
//
// STORE_INTERVAL интерпретируется как целое число секунд. Значение 0 включает синхронное сохранение
// (после каждого обновления метрики); положительное значение включает периодическое сохранение
// с указанным интервалом.
// FILE_STORAGE_PATH — путь до файла, используемого для сохранения метрик.
// RESTORE определяет, нужно ли загружать сохранённые метрики из FILE_STORAGE_PATH при старте.
type ServiceConfig struct {
	RunAddr         *string `env:"ADDRESS"`
	StoreInterval   *int    `env:"STORE_INTERVAL"`
	FileStoragePath *string `env:"FILE_STORAGE_PATH"`
	Restore         *bool   `env:"RESTORE"`
}
