package model

// URLRecord — запись о сокращённом URL.
type URLRecord struct {
	ID          string
	OriginalURL string
	UserID      string
	Deleted     bool // флаг мягкого удаления
}

// StorageRecord — запись для сериализации в JSON-файл.
type StorageRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	Deleted     bool   `json:"is_deleted"`
}
