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

// FileStorage persists the in-memory metrics snapshot to a JSON file.
//
// It writes through a temporary file followed by an atomic rename.
// This prevents partially written JSON if the process crashes/restarts mid-save:
// on startup Restore will see either the previous valid snapshot or the new one.
type FileStorage struct {
	repo *MemoryRepo
	path string
}

// NewFileStorage creates a file-backed snapshot storage for the given repository.
func NewFileStorage(repo *MemoryRepo, path string) *FileStorage {
	if path == "" {
		panic("file storage path is empty")
	}
	return &FileStorage{
		repo: repo,
		path: path,
	}
}

// Save persists the current metrics snapshot to disk.
func (s *FileStorage) Save(_ context.Context) error {
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
	// Encode marshals to JSON in a streaming fashion, writing directly to the file.
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

func applyMetricsSnapshot(repo *MemoryRepo, metrics []models.Metrics) {
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			repo.SetGauge(context.Background(), m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			repo.SetCounter(m.ID, *m.Delta)
		}
	}
}

func (s *FileStorage) snapshot() []models.Metrics {
	gauges := s.repo.Gauges(context.Background())
	counters := s.repo.Counters(context.Background())

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
