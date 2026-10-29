package audit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	httpQueueSize      = 1024
	httpWorkerCount    = 4
	httpMaxRetries     = 5
	httpInitialBackoff = 100 * time.Millisecond
)

// HTTPObserver отправляет события на удалённый сервер методом POST.
// Внутри использует ограниченную очередь и пул воркеров: Notify только
// кладёт событие в очередь, а отправка происходит асинхронно.
type HTTPObserver struct {
	client *http.Client
	url    string
	events chan Event
	done   chan struct{}
	wg     sync.WaitGroup
}

// NewHTTPObserver создаёт наблюдателя для удалённого приёмника
// и запускает пул воркеров.
func NewHTTPObserver(url string) *HTTPObserver {
	o := &HTTPObserver{
		client: &http.Client{Timeout: 5 * time.Second},
		url:    url,
		events: make(chan Event, httpQueueSize),
		done:   make(chan struct{}),
	}
	for i := 0; i < httpWorkerCount; i++ {
		o.wg.Add(1)
		go o.worker()
	}
	return o
}

// Notify кладёт событие в очередь, не блокируя вызывающий код.
// Если очередь переполнена — событие теряется с предупреждением в лог.
func (o *HTTPObserver) Notify(event Event) {
	select {
	case o.events <- event:
	case <-o.done:
	default:
		slog.Warn("audit http queue full, dropping event", "url", o.url)
	}
}

// Close останавливает воркеров, дожидаясь отправки уже принятых событий.
// Повторный вызов не паникует.
func (o *HTTPObserver) Close() error {
	select {
	case <-o.done:
	default:
		close(o.done)
	}
	o.wg.Wait()
	return nil
}

func (o *HTTPObserver) worker() {
	defer o.wg.Done()
	for {
		select {
		case event := <-o.events:
			o.sendWithRetry(event)
		case <-o.done:
			for {
				select {
				case event := <-o.events:
					o.sendWithRetry(event)
				default:
					return
				}
			}
		}
	}
}

func (o *HTTPObserver) sendWithRetry(event Event) {
	backoff := httpInitialBackoff
	for attempt := 1; attempt <= httpMaxRetries; attempt++ {
		select {
		case <-o.done:
			return
		default:
		}

		ok, retry := o.sendOnce(event)
		if ok {
			return
		}
		if !retry {
			return
		}
		if attempt == httpMaxRetries {
			slog.Warn("audit http giving up after retries", "url", o.url, "attempts", attempt)
			return
		}
		select {
		case <-time.After(backoff):
			backoff *= 2
		case <-o.done:
			return
		}
	}
}

// sendOnce делает одну попытку отправки.
// Возвращает:
//   - ok — успешная доставка (2xx);
//   - retry — имеет ли смысл повторять (сетевые ошибки, 5xx).
func (o *HTTPObserver) sendOnce(event Event) (ok, retry bool) {
	buf := getBuf()
	defer putBuf(buf)

	if err := json.NewEncoder(buf).Encode(event); err != nil {
		slog.Error("audit http encode failed", "error", err)
		return false, false
	}

	resp, err := o.client.Post(o.url, "application/json", buf)
	if err != nil {
		slog.Warn("audit http post failed", "url", o.url, "error", err)
		return false, true
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, false
	}

	slog.Warn("audit http non-2xx response", "url", o.url, "status", resp.StatusCode)
	return false, resp.StatusCode >= 500
}
