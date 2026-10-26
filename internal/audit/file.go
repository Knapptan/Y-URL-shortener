package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

// FileObserver пишет события аудита в файл (append, одна строка на событие).
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileObserver открывает файл для дозаписи.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if err := ensureTrailingNewline(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &FileObserver{file: f}, nil
}

// ensureTrailingNewline гарантирует, что новое событие начнётся с новой строки.
func ensureTrailingNewline(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}
	last := make([]byte, 1)
	if _, err := f.ReadAt(last, info.Size()-1); err != nil {
		return err
	}
	if last[0] == '\n' {
		return nil
	}
	_, err = f.Write([]byte{'\n'})
	return err
}

// Notify сериализует событие и дописывает его в файл.
func (o *FileObserver) Notify(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("audit file marshal failed", "error", err)
		return
	}
	data = append(data, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()
	if _, err := o.file.Write(data); err != nil {
		slog.Error("audit file write failed", "error", err)
	}
}

// Close закрывает файл.
func (o *FileObserver) Close() error {
	return o.file.Close()
}
