// Package audit реализует паттерн Observer для аудита успешного приёма метрик.
package audit

//go:generate go run ../../cmd/reset

// generate:reset
// Event — запись аудита после успешного приёма метрик.
type Event struct {
	// TS — Unix-время события (секунды).
	TS int64 `json:"ts"`
	// Metrics — имена обновлённых метрик.
	Metrics []string `json:"metrics"`
	// IPAddress — IP клиента из HTTP-запроса.
	IPAddress string `json:"ip_address"`
}
