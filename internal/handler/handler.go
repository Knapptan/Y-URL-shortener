package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

// URLHandler содержит сервис и базовый URL.
type URLHandler struct {
	service *service.URLService
	baseURL string
}

type shortenRequest struct {
	URL string `json:"url"`
}

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

type userURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// NewURLHandler создаёт новый обработчик.
func NewURLHandler(svc *service.URLService, baseURL string) *URLHandler {
	return &URLHandler{service: svc, baseURL: baseURL}
}

// getUserID извлекает userID из контекста.
func getUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	return userID, ok && userID != ""
}

// CreateShortURL обрабатывает POST /.
func (h *URLHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, exists, err := h.service.Shorten(string(body), userID)
	if err != nil {
		slog.Error("Failed to shorten URL", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if exists {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(h.baseURL + "/" + id))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(h.baseURL + "/" + id))
}

// RedirectToOriginal обрабатывает GET /{id}.
func (h *URLHandler) RedirectToOriginal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = strings.TrimPrefix(r.URL.Path, "/")
	}
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	originalURL, ok, deleted, err := h.service.GetOriginal(id)
	if err != nil {
		slog.Error("Failed to get original URL", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}
	if deleted {
		http.Error(w, "Gone", http.StatusGone)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// CreateShortenJSON обрабатывает POST /api/shorten.
func (h *URLHandler) CreateShortenJSON(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "URL is empty", http.StatusBadRequest)
		return
	}

	id, exists, err := h.service.Shorten(req.URL, userID)
	if err != nil {
		slog.Error("Failed to shorten URL", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := shortenResponse{Result: h.baseURL + "/" + id}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(resp); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if exists {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}

// CreateShortenBatch обрабатывает POST /api/shorten/batch.
func (h *URLHandler) CreateShortenBatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	items := make([]service.BatchItem, len(req))
	for i, v := range req {
		items[i] = service.BatchItem{
			CorrelationID: v.CorrelationID,
			OriginalURL:   v.OriginalURL,
		}
	}

	results, err := h.service.ShortenBatch(items, userID)
	if err != nil {
		slog.Error("Failed to process batch", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

// GetUserURLs обрабатывает GET /api/user/urls.
func (h *URLHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	records, err := h.service.GetUserURLs(userID)
	if err != nil {
		slog.Error("Failed to get user URLs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if len(records) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]userURLResponse, len(records))
	for i, rec := range records {
		resp[i] = userURLResponse{
			ShortURL:    h.baseURL + "/" + rec.ID,
			OriginalURL: rec.OriginalURL,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

// DeleteUserURLs обрабатывает DELETE /api/user/urls.
func (h *URLHandler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ids []string
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(ids) == 0 {
		http.Error(w, "Empty IDs array", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteURLs(userID, ids); err != nil {
		slog.Error("Failed to delete URLs", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
