package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"go.uber.org/zap"
)

// HTTPObserver — наблюдатель, который отправляет события аудита POST-запросом.
// Реализует интерфейс Observer.
type HTTPObserver struct {
	url    string       // полный URL из --audit-url / AUDIT_URL
	client *http.Client // HTTP-клиент с таймаутом
	log    *zap.Logger
}

// NewHTTPObserver создаёт observer, который шлет события на удалённый сервер.
func NewHTTPObserver(url string, log *zap.Logger) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second, // не висим, если observer недоступен
		},
		log: logger.OrNop(log),
	}
}

// Notify сериализует event в JSON и отправляет POST на audit-url.
func (h *HTTPObserver) Notify(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		h.log.Error("audit http: marshal event", zap.Error(err))
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		h.log.Error("audit http: new request", zap.Error(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		h.log.Error("audit http: do request", zap.String("url", h.url), zap.Error(err))
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("unexpected status %d", resp.StatusCode)
		h.log.Error("audit http: bad status",
			zap.String("url", h.url),
			zap.Int("status", resp.StatusCode),
		)
		return err
	}
	return nil
}
