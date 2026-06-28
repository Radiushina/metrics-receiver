package audit

import "context"

// Publisher - издатель, регистрирует observers и вызывает Notify.
// Регистрирует observers и рассылает им Event после успешной обработки метрик.
type Publisher struct {
	observers []Observer
}

// NewPublisher создаёт издателя без подписчиков.
// Подписчиков  добавляем через Register в main, если заданы --audit-file / --audit-url.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Register добавляет observer в список подписчиков.
// Вызывается при старте сервера для каждого подписчика (файл, URL).
func (p *Publisher) Register(o Observer) {
	if o == nil {
		return
	}
	p.observers = append(p.observers, o)
}

// Enabled сообщает, есть ли хотя бы один приёмник.
// Удобно в handler: if h.audit != nil && h.audit.Enabled() { ... }
func (p *Publisher) Enabled() bool {
	return len(p.observers) > 0
}

// Notify рассылает событие всем зарегистрированным observers.
// Вызывается из handler ПОСЛЕ успешного update + persist.
func (p *Publisher) Notify(ctx context.Context, e Event) {
	for _, o := range p.observers {
		_ = o.Notify(ctx, e)
	}
}
