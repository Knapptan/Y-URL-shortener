package audit

import (
	"bytes"
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
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("audit http marshal failed", "error", err)
		return
	}

	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(data))
	if err != nil {
		slog.Error("audit http post failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		slog.Warn("audit http unexpected status", "status", resp.StatusCode)
	}
}
