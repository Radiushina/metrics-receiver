package audit

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	"go.uber.org/zap"
)

// FileObserver — наблюдатель, который пишет события аудита в файл.
// Реализует интерфейс Observer.
// Каждое событие — одна строка JSON + перевод строки (append в конец файла).
type FileObserver struct {
	path string
	logg *zap.Logger
	mu   sync.Mutex
}

// NewFileObserver создаёт observer, пишущий в указанный файл.
func NewFileObserver(path string, logg *zap.Logger) *FileObserver {
	return &FileObserver{
		path: path,
		logg: logger.OrNop(logg),
	}
}

// Notify сериализует event в JSON и дописывает строку в конец файла.
// Вызывается Publisher'ом; ctx здесь можно не использовать (запись локальная).
func (f *FileObserver) Notify(_ context.Context, event Event) error {
	// Event -> JSON в одну строку
	data, err := json.Marshal(event)
	if err != nil {
		f.logg.Error("audit file: marshal event", zap.Error(err))
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	// O_APPEND — всегда в конец; O_CREATE — создать файл, если нет; O_WRONLY — только запись
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		f.logg.Error("audit file: open", zap.String("path", f.path), zap.Error(err))
		return err
	}
	defer func() { _ = file.Close() }()

	// Пишем в файл JSON + '\n' — по одному событию на строку
	if _, err := file.Write(append(data, '\n')); err != nil {
		f.logg.Error("audit file: write", zap.String("path", f.path), zap.Error(err))
		return err
	}

	return nil
}
