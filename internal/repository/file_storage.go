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
// Запись выполняется через временный файл с последующим атомарным rename.
// Это защищает от ситуации,
// когда процесс падает/перезапускается во время сохранения:
// после старта RESTORE увидит либо
// предыдущий корректный снимок, либо новый, но не частично записанный JSON.
type FileStorage struct {
	repo *Repository
	path string
}

// NewFileStorage creates a file-backed snapshot storage for the given repository.
func NewFileStorage(repo *Repository, path string) *FileStorage {
	return &FileStorage{
		repo: repo,
		path: path,
	}
}

// Save persists the current metrics snapshot to disk.
func (s *FileStorage) Save(_ context.Context) error {
	if s.path == "" {
		return errors.New("file storage path is empty")
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
	// committed indicates that tmpPath has been atomically moved to the final path via Rename.
	// Until then we must clean up (close + remove) the temp file on any error; after Rename,
	// removing tmpPath would delete the final file.
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = f.Close()
		_ = os.Remove(tmpPath)
	}()

	enc := json.NewEncoder(f)
	// Encode — это и есть маршалинг в JSON, только потоковый: без промежуточного []byte пишем сразу в файл.
	enc.SetIndent("", "  ")
	if err := enc.Encode(metrics); err != nil {
		return fmt.Errorf("encode metrics: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	committed = true

	return nil
}

// Restore loads metrics from disk into the repository if the file exists.
func (s *FileStorage) Restore(_ context.Context) error {
	if s.path == "" {
		return errors.New("file storage path is empty")
	}

	f, ok, err := openStorageFile(s.path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	defer func() { _ = f.Close() }()

	b, empty, err := readAllOrEmpty(f)
	if err != nil {
		return err
	}
	if empty {
		return nil
	}

	metrics, err := decodeMetrics(b)
	if err != nil {
		return err
	}
	applyMetricsSnapshot(s.repo, metrics)
	return nil
}

func openStorageFile(path string) (f *os.File, ok bool, err error) {
	f, err = os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("open storage file: %w", err)
	}
	return f, true, nil
}

func readAllOrEmpty(r io.Reader) (b []byte, empty bool, err error) {
	b, err = io.ReadAll(r)
	if err != nil {
		return nil, false, fmt.Errorf("read storage file: %w", err)
	}
	return b, len(b) == 0, nil
}

func decodeMetrics(b []byte) ([]models.Metrics, error) {
	var metrics []models.Metrics
	if err := json.Unmarshal(b, &metrics); err != nil {
		return nil, fmt.Errorf("decode metrics: %w", err)
	}
	return metrics, nil
}

func applyMetricsSnapshot(repo *Repository, metrics []models.Metrics) {
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			repo.SetGauge(m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			repo.SetCounter(m.ID, *m.Delta)
		}
	}
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
