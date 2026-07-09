package audit

import (
	"context"
	"sync"

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"go.uber.org/zap"
)

const auditWorkers = 2

// Publisher — издатель: регистрирует observers и асинхронно рассылает им Event.
type Publisher struct {
	observers []Observer
	jobs      chan Event
	log       *zap.Logger
	closeOnce sync.Once
}

// NewPublisher создаёт издателя с пулом воркеров для доставки событий аудита.
// Подписчиков добавляем через Register в main, если заданы --audit-file / --audit-url.
func NewPublisher(ctx context.Context, log *zap.Logger) *Publisher {
	p := &Publisher{
		jobs: make(chan Event, auditWorkers*2),
		log:  logger.OrNop(log),
	}
	for range auditWorkers {
		go p.worker(ctx)
	}
	return p
}

// Register добавляет observer в список подписчиков.
func (p *Publisher) Register(o Observer) {
	if o == nil {
		return
	}
	p.observers = append(p.observers, o)
}

// Enabled сообщает, есть ли хотя бы один приёмник.
func (p *Publisher) Enabled() bool {
	return len(p.observers) > 0
}

// Notify ставит событие в очередь и сразу возвращает управление HTTP-обработчику.
func (p *Publisher) Notify(_ context.Context, e Event) {
	select {
	case p.jobs <- e:
	default:
		p.log.Warn("audit: очередь переполнена, событие пропущено")
	}
}

// Close останавливает воркеров после обработки оставшихся событий в очереди.
func (p *Publisher) Close() {
	p.closeOnce.Do(func() { close(p.jobs) })
}

func (p *Publisher) worker(ctx context.Context) {
	for event := range p.jobs {
		p.notifyObservers(ctx, event)
	}
}

func (p *Publisher) notifyObservers(ctx context.Context, event Event) {
	var wg sync.WaitGroup
	for _, o := range p.observers {
		wg.Go(func() {
			if err := o.Notify(ctx, event); err != nil {
				p.log.Error("audit observer failed", zap.Error(err))
			}
		})
	}
	wg.Wait()
}
