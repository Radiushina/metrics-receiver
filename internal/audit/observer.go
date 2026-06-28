package audit

import "context"

// Observer - интерфейс наблюдателя (приемник аудита).
// Каждая реализация знает, куда отправить событие: в файл, по HTTP и т.д.
type Observer interface {
	// Notify вызывается Publisher'ом после успешного приёма метрик.
	// ctx — для отмены/таймаута HTTP-запроса.
	// Возвращает ошибку, если запись/отправка не удалась.
	// Publisher не должен из-за этой ошибки ломать ответ клиенту — только залогировать.
	Notify(ctx context.Context, event Event) error
}
