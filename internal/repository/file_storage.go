package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

// FileStorage сохраняет текущий снимок метрик из памяти в JSON-файл.
//
// Запись выполняется через временный файл с последующим атомарным rename. Это защищает от ситуации,
// когда процесс падает/перезапускается во время сохранения: после старта RESTORE увидит либо
// предыдущий корректный снимок, либо новый, но не частично записанный JSON.
type FileStorage struct {
	repo *Repository
	path string
}

func NewFileStorage(repo *Repository, path string) *FileStorage {
	return &FileStorage{
		repo: repo,
		path: path,
	}
}

func (s *FileStorage) Save(_ context.Context) error {
	if s.path == "" {
		return fmt.Errorf("file storage path is empty")
	}

	dir := filepath.Dir(s.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create storage dir: %w", err)
		}
	}

	metrics := s.snapshot()

	tmpPath := s.path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	enc := json.NewEncoder(f)
	// Encode — это и есть маршалинг в JSON, только потоковый: без промежуточного []byte пишем сразу в файл.
	enc.SetIndent("", "  ")
	if err := enc.Encode(metrics); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("encode metrics: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (s *FileStorage) Restore(_ context.Context) error {
	if s.path == "" {
		return fmt.Errorf("file storage path is empty")
	}

	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open storage file: %w", err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read storage file: %w", err)
	}
	if len(b) == 0 {
		return nil
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(b, &metrics); err != nil {
		return fmt.Errorf("decode metrics: %w", err)
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			s.repo.SetGauge(m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			// For persistence/restoration, Delta stores the current absolute counter value.
			s.repo.SetCounter(m.ID, *m.Delta)
		default:
			continue
		}
	}

	return nil
}

func (s *FileStorage) snapshot() []models.Metrics {
	gauges := s.repo.Gauges()
	counters := s.repo.Counters()

	out := make([]models.Metrics, 0, len(gauges)+len(counters))

	for id, v := range gauges {
		val := v
		out = append(out, models.Metrics{
			ID:    id,
			MType: models.Gauge,
			Value: &val,
		})
	}

	for id, v := range counters {
		delta := v
		out = append(out, models.Metrics{
			ID:    id,
			MType: models.Counter,
			Delta: &delta,
		})
	}

	return out
}
