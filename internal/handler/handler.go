package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/go-chi/chi"
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

	id, err := h.service.Shorten(originalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	id, err := h.service.Shorten(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Формируем ответ
	resp := shortenResponse{
		Result: h.baseURL + "/" + id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
