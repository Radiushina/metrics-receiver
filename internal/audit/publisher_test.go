package audit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type slowObserver struct {
	delay time.Duration
	calls atomic.Int32
}

func (o *slowObserver) Notify(_ context.Context, _ Event) error {
	o.calls.Add(1)
	time.Sleep(o.delay)
	return nil
}

func TestPublisher_Notify_DoesNotBlockOnSlowObserver(t *testing.T) {
	pub := NewPublisher(context.Background(), zap.NewNop())
	t.Cleanup(pub.Close)
	slow := &slowObserver{delay: 200 * time.Millisecond}
	fast := &slowObserver{}
	pub.Register(slow)
	pub.Register(fast)

	start := time.Now()
	pub.Notify(context.Background(), Event{TS: 1, Metrics: []string{"a"}, IPAddress: "127.0.0.1"})
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("Notify заблокировался на %v", elapsed)
	}

	deadline := time.Now().Add(time.Second)
	for slow.calls.Load() == 0 || fast.calls.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatalf("observers не вызваны: slow=%d fast=%d", slow.calls.Load(), fast.calls.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestPublisher_Notify_ObserversRunInParallel(t *testing.T) {
	pub := NewPublisher(context.Background(), zap.NewNop())
	t.Cleanup(pub.Close)

	var mu sync.Mutex
	active := 0
	maxActive := 0

	track := func(_ context.Context, _ Event) error {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return nil
	}

	pub.Register(ObserverFunc(track))
	pub.Register(ObserverFunc(track))

	pub.Notify(context.Background(), Event{TS: 1})
	time.Sleep(200 * time.Millisecond)

	if maxActive < 2 {
		t.Fatalf("ожидали параллельный вызов observers, maxActive=%d", maxActive)
	}
}

// ObserverFunc адаптирует функцию к интерфейсу Observer для тестов.
type ObserverFunc func(context.Context, Event) error

func (f ObserverFunc) Notify(ctx context.Context, e Event) error {
	return f(ctx, e)
}
