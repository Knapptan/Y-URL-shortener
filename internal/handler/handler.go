package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/go-chi/chi"
)

// URLHandler содержит сервис и методы-обработчики.
type URLHandler struct {
	service *service.URLService
	baseURL string // базовый адрес для формирования коротких URL
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
