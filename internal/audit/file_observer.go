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
	file *os.File
	logg *zap.Logger
	mu   sync.Mutex
}

// NewFileObserver создаёт observer, пишущий в указанный файл.
func NewFileObserver(path string, logg *zap.Logger) (*FileObserver, error) {
	logg = logger.OrNop(logg)
	// O_APPEND — всегда в конец; O_CREATE — создать файл, если нет; O_WRONLY — только запись
	// Открываем файл один раз и закроем только когда программа завершится.
	// Файл держим открытым так как пишем в него постоянно.
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logg.Error("audit file: open", zap.String("path", path), zap.Error(err))
		return nil, err
	}
	return &FileObserver{
		path: path,
		file: file,
		logg: logg,
	}, nil
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

	// Пишем в файл JSON + '\n' — по одному событию на строку
	if _, err := f.file.Write(append(data, '\n')); err != nil {
		f.logg.Error("audit file: write", zap.String("path", f.path), zap.Error(err))
		return err
	}

	return nil
}

// Close закрывает файл аудита при завершении работы сервера.
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}
