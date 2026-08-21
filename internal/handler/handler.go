package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

// URLHandler - содержит сервис и методы-обработчики.
type URLHandler struct {
	service *service.URLService
	baseURL string // базовый адрес для формирования коротких URL
}

// shortenRequest - структура запроса.
type shortenRequest struct {
	URL string `json:"url"`
}

// shortenResponse - структура ответа.
type shortenResponse struct {
	Result string `json:"result"`
}

type batchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type batchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// NewURLHandler конструктор.
func NewURLHandler(svc *service.URLService, baseURL string) *URLHandler {
	return &URLHandler{
		service: svc,
		baseURL: baseURL,
	}
}

// CreateShortURL обрабатывает POST /.
func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}
	originalURL := string(body)

	id, exists, err := h.service.Shorten(originalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if exists {
		// Возвращаем 409 Conflict с уже существующим URL
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(h.baseURL + "/" + id))
		return
	}
	// формируем полную короткую ссылку
	shortURL := h.baseURL + "/" + id

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated) // 201
	w.Write([]byte(shortURL))
}

// RedirectToOriginal обрабатывает GET /{id}.
func (h *URLHandler) RedirectToOriginal(w http.ResponseWriter, r *http.Request) {
	// Попытка получить ID из параметра chi (если используется роутер)
	id := chi.URLParam(r, "id")
	// Если параметр пуст (тесты, прямой вызов), берём из пути
	if id == "" {
		id = strings.TrimPrefix(r.URL.Path, "/")
	}

	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	originalURL, ok := h.service.GetOriginal(id)
	if !ok {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// CreateShortenJSON - обрабатывает POST /api/shorten.
func (h *URLHandler) CreateShortenJSON(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод (хотя chi сам отфильтрует, но оставим для надёжности)
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем JSON
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.URL == "" {
		http.Error(w, "URL is empty", http.StatusBadRequest)
		return
	}

	// Вызываем сервис
	id, exists, err := h.service.Shorten(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if exists {
		resp := shortenResponse{Result: h.baseURL + "/" + id}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("Failed to encode response", "error", err)
		}
		return
	}

	// Формируем ответ
	resp := shortenResponse{
		Result: h.baseURL + "/" + id,
	}

	// Кодируем в буфер
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(resp); err != nil {
		// Если не удалось закодировать даже в память – ошибка сервера
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Теперь отправляем заголовки и тело
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(buf.Bytes()); err != nil {
		// ошибка записи клиенту – можем только залогировать
		slog.Error("Failed to write response", "error", err)
	}
}

func (h *URLHandler) CreateShortenBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req []batchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	// Преобразуем в сервисные объекты
	items := make([]service.BatchItem, len(req))
	for i, v := range req {
		items[i] = service.BatchItem{
			CorrelationID: v.CorrelationID,
			OriginalURL:   v.OriginalURL,
		}
	}

	results, err := h.service.ShortenBatch(items)
	if err != nil {
		// Если конфликт ID, можно вернуть 409, но пока 400
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Формируем ответ
	resp := make([]batchResponse, len(results))
	for i, v := range results {
		resp[i] = batchResponse{
			CorrelationID: v.CorrelationID,
			ShortURL:      v.ShortURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
