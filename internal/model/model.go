package model

// StorageRecord - запись для сохранения в мапу.
type URLRecord struct {
	ID          string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// StorageRecord - запись для сохранения в JSON-файл.
type StorageRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
