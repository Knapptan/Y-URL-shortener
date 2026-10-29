package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

const fileQueueSize = 1024

// FileObserver пишет события аудита в файл (append, одна строка на событие).
type FileObserver struct {
	file   *os.File
	events chan Event
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewFileObserver открывает файл для дозаписи и запускает воркера.
// Если файл уже существует и не заканчивается на '\n', в конец
// дописывается '\n', чтобы новое событие не приклеилось к предыдущей строке.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		buf := make([]byte, 1)
		if _, err := f.ReadAt(buf, info.Size()-1); err == nil && buf[0] != '\n' {
			if _, err := f.Write([]byte("\n")); err != nil {
				_ = f.Close()
				return nil, err
			}
		}
	}

	o := &FileObserver{
		file:   f,
		events: make(chan Event, fileQueueSize),
		done:   make(chan struct{}),
	}
	o.wg.Add(1)
	go o.worker()
	return o, nil
}

// Notify кладёт событие в очередь. Не блокирует вызывающий код.
func (o *FileObserver) Notify(event Event) {
	select {
	case o.events <- event:
	case <-o.done:
	default:
		slog.Warn("audit file queue full, dropping event")
	}
}

// Close останавливает воркера, дожидаясь записи уже принятых событий,
// затем закрывает файл.
func (o *FileObserver) Close() error {
	select {
	case <-o.done:
	default:
		close(o.done)
	}
	o.wg.Wait()
	return o.file.Close()
}

func (o *FileObserver) worker() {
	defer o.wg.Done()
	for {
		select {
		case event := <-o.events:
			o.write(event)
		case <-o.done:
			for {
				select {
				case event := <-o.events:
					o.write(event)
				default:
					return
				}
			}
		}
	}
}

// write вызывается только из worker — конкурентного доступа нет.
func (o *FileObserver) write(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("audit file marshal failed", "error", err)
		return
	}
	data = append(data, '\n')
	if _, err := o.file.Write(data); err != nil {
		slog.Error("audit file write failed", "error", err)
	}
}
