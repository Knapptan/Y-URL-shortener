package model

// URLRecord — запись о сокращённом URL.
type URLRecord struct {
	ID          string
	OriginalURL string
	UserID      string
	Deleted     bool
}

// StorageRecord — запись для сериализации в JSON-файл.
type StorageRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	Deleted     bool   `json:"is_deleted"`
}

// JobStatus — состояние асинхронной задачи.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusCompleted JobStatus = "completed"
)

// AsyncJob — задача асинхронного сокращения URL.
type AsyncJob struct {
	CorrelationID string
	URL           string
	UserID        string
	ShortID       string
	ShortURL      string
	Status        JobStatus
	Error         string // если задача завершилась ошибкой
}
