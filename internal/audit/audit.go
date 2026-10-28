// Package audit реализует паттерн «Наблюдатель» для аудита запросов.
package audit

import (
	"sync"
	"time"
)

// Константы действий аудита.
const (
	// ActionShorten — действие «создание короткого URL».
	ActionShorten = "shorten"
	// ActionFollow — действие «переход по короткой ссылке».
	ActionFollow = "follow"
)

// Event — событие аудита.
type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

// Observer — приёмник аудита.
type Observer interface {
	Notify(event Event)
}

// Publisher уведомляет всех зарегистрированных наблюдателей.
type Publisher struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewPublisher создаёт издателя без наблюдателей.
func NewPublisher() *Publisher {
	return &Publisher{observers: make([]Observer, 0)}
}

// Register добавляет наблюдателя.
func (p *Publisher) Register(o Observer) {
	if o == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, o)
}

// Publish отправляет событие всем наблюдателям.
func (p *Publisher) Publish(event Event) {
	if p == nil {
		return
	}
	p.mu.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()

	for _, o := range observers {
		o.Notify(event)
	}
}

// NewEvent формирует событие с текущим timestamp.
func NewEvent(action, userID, url string) Event {
	return Event{
		TS:     time.Now().Unix(),
		Action: action,
		UserID: userID,
		URL:    url,
	}
}
