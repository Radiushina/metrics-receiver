// Package config описывает конфигурацию сервера и агента из переменных окружения.
package config

// AgentConfig — опциональные значения из переменных окружения для агента метрик.
// Поля-указатели равны nil, если соответствующая переменная не задана.
// POLL_INTERVAL и REPORT_INTERVAL задаются целым числом секунд.
type AgentConfig struct {
	// ADDRESS — хост:порт HTTP-сервера приёма метрик (без схемы; к адресу добавляется http://).
	// При наличии в окружении перекрывает значение флага -a.
	RunAddr *string `env:"ADDRESS"`
	// POLL_INTERVAL — как часто опрашивать runtime.MemStats и обновлять gauge (секунды).
	// При наличии в окружении перекрывает значение флага -p.
	PollIntervalSec *int64 `env:"POLL_INTERVAL"`
	// REPORT_INTERVAL — как часто отправлять метрики на сервер (секунды).
	// При наличии в окружении перекрывает значение флага -r.
	ReportIntervalSec *int64 `env:"REPORT_INTERVAL"`
	// Key — секрет для подписи тел (HashSHA256, HMAC-SHA256); если не задан — без подписи.
	// При наличии в окружении перекрывает значение флага -k.
	Key *string `env:"KEY"`
	// RateLimit - количество одновременно исходящих запросов на сервер
	RateLimit *int64 `env:"RATE_LIMIT"`
	// CryptoKey - путь до файла с публичным ключем
	CryptoKey *string `env:"CRYPTO_KEY"`
}

// ServiceConfig содержит опциональные значения из переменных окружения
// для конфигурации сервера.
// Поля-указатели равны nil, если соответствующая переменная окружения
// не задана.
type ServiceConfig struct {
	// ADDRESS — адрес и порт, на которых слушает HTTP-сервер (перекрывает значение
	// по умолчанию и совпадает по смыслу с флагом командной строки -a).
	RunAddr *string `env:"ADDRESS"`
	// DATABASE_DSN — строка подключения к PostgreSQL (DSN); при наличии переменной
	// перекрывает значение флага -d.
	// Локально с docker compose: postgres://developer:my_pass@localhost:5432/metrics?sslmode=disable
	DbDsn *string `env:"DATABASE_DSN"`
	// STORE_INTERVAL интерпретируется как целое число секунд. Значение 0 включает
	// синхронное сохранение
	// (после каждого обновления метрики); положительное значение включает
	// периодическое сохранение
	// с указанным интервалом.
	StoreInterval *int `env:"STORE_INTERVAL"`
	// FILE_STORAGE_PATH — путь до файла, используемого для сохранения метрик.
	FileStoragePath *string `env:"FILE_STORAGE_PATH"`
	// RESTORE определяет, нужно ли загружать сохранённые метрики из
	// FILE_STORAGE_PATH при старте.
	Restore *bool `env:"RESTORE"`
	// Key — секрет для подписи тел (HashSHA256, HMAC-SHA256); если не задан — без подписи.
	Key *string `env:"KEY"`
	// AUDIT_FILE - путь к файлу, в который сохраняются логи аудита. Если параметр не передан, аудит должен быть отключен
	AuditFilePath *string `env:"AUDIT_FILE"`
	// AUDIT_URL - полный URL, по которому отправляются логи аудита. Если параметр не передан, аудит должен быть отключен
	AuditURL *string `env:"AUDIT_URL"`
	// CryptoKey - путь до файла с приватным ключем
	CryptoKey *string `env:"CRYPTO_KEY"`
}
