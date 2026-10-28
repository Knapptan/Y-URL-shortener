package audit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// HTTPObserver отправляет события на удалённый сервер методом POST.
type HTTPObserver struct {
	client *http.Client
	url    string
}

// NewHTTPObserver создаёт наблюдателя для удалённого приёмника.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		client: &http.Client{Timeout: 5 * time.Second},
		url:    url,
	}
}

// Notify сериализует событие и отправляет его на удалённый сервер.
func (o *HTTPObserver) Notify(event Event) {
	go func() {
		buf := getBuf()
		defer putBuf(buf)

		if err := json.NewEncoder(buf).Encode(event); err != nil {
			slog.Error("audit http encode failed", "error", err)
			return
		}
		resp, err := o.client.Post(o.url, "application/json", buf)
		if err != nil {
			slog.Error("audit http post failed", "error", err)
			return
		}
		resp.Body.Close()
	}()
}
