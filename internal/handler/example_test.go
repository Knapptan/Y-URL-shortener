package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// exampleRepo — детерминированный in-memory репозиторий для примеров.
type exampleRepo struct {
	urls map[string]model.URLRecord
}

func newExampleRepo() *exampleRepo {
	return &exampleRepo{urls: make(map[string]model.URLRecord)}
}

func (r *exampleRepo) Save(id string, record model.URLRecord) error {
	record.ID = id
	r.urls[id] = record
	return nil
}

func (r *exampleRepo) Get(id string) (model.URLRecord, bool, error) {
	rec, ok := r.urls[id]
	return rec, ok, nil
}

func (r *exampleRepo) SaveBatch(batch map[string]model.URLRecord) error {
	for id, rec := range batch {
		rec.ID = id
		r.urls[id] = rec
	}
	return nil
}

func (r *exampleRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	for id, rec := range r.urls {
		if rec.OriginalURL == originalURL {
			return id, rec, true, nil
		}
	}
	return "", model.URLRecord{}, false, nil
}

func (r *exampleRepo) GetByUserID(userID string) ([]model.URLRecord, error) {
	var result []model.URLRecord
	for _, rec := range r.urls {
		if rec.UserID == userID {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (r *exampleRepo) BatchDeleteByUserID(userID string, ids []string) error {
	for _, id := range ids {
		if rec, ok := r.urls[id]; ok && rec.UserID == userID {
			rec.Deleted = true
			r.urls[id] = rec
		}
	}
	return nil
}

// withUserID эмулирует middleware.AuthMiddleware.
func withUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID))
}

// ExampleURLHandler_CreateShortURL демонстрирует POST /.
// Если URL уже существует, сервер вернёт 409 Conflict с готовой ссылкой.
func ExampleURLHandler_CreateShortURL() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{
		OriginalURL: "https://practicum.yandex.ru",
		UserID:      "user-1",
	})

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru"))
	req.Header.Set("Content-Type", "text/plain")
	req = withUserID(req, "user-1")

	w := httptest.NewRecorder()
	h.CreateShortURL(w, req)

	fmt.Println("status:", w.Code)
	fmt.Println("body:", strings.TrimSpace(w.Body.String()))
	// Output:
	// status: 409
	// body: http://localhost:8080/abc123
}

// ExampleURLHandler_CreateShortenJSON демонстрирует POST /api/shorten.
func ExampleURLHandler_CreateShortenJSON() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{
		OriginalURL: "https://practicum.yandex.ru",
		UserID:      "user-1",
	})

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	body := `{"url":"https://practicum.yandex.ru"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")

	w := httptest.NewRecorder()
	h.CreateShortenJSON(w, req)

	var resp struct {
		Result string `json:"result"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	fmt.Println("status:", w.Code)
	fmt.Println("result:", resp.Result)
	// Output:
	// status: 409
	// result: http://localhost:8080/abc123
}

// ExampleURLHandler_RedirectToOriginal демонстрирует GET /{id}.
func ExampleURLHandler_RedirectToOriginal() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{
		OriginalURL: "https://practicum.yandex.ru",
		UserID:      "user-1",
	})

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()
	h.RedirectToOriginal(w, req)

	fmt.Println("status:", w.Code)
	fmt.Println("location:", w.Header().Get("Location"))
	// Output:
	// status: 307
	// location: https://practicum.yandex.ru
}

// ExampleURLHandler_GetUserURLs_empty демонстрирует GET /api/user/urls
// при отсутствии URL у пользователя — ответ 204 No Content.
func ExampleURLHandler_GetUserURLs_empty() {
	repo := newExampleRepo()
	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = withUserID(req, "user-1")

	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	fmt.Println("status:", w.Code)
	// Output:
	// status: 204
}

// ExampleURLHandler_GetUserURLs демонстрирует GET /api/user/urls
// с непустым списком URL пользователя.
func ExampleURLHandler_GetUserURLs() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{
		OriginalURL: "https://practicum.yandex.ru",
		UserID:      "user-1",
	})
	_ = repo.Save("def456", model.URLRecord{
		OriginalURL: "https://ya.ru",
		UserID:      "user-2", // чужой URL не должен попасть в ответ
	})

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = withUserID(req, "user-1")

	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	var resp []struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	fmt.Println("status:", w.Code)
	fmt.Println("count:", len(resp))
	fmt.Println("original:", resp[0].OriginalURL)
	// Output:
	// status: 200
	// count: 1
	// original: https://practicum.yandex.ru
}

// ExampleURLHandler_DeleteUserURLs демонстрирует DELETE /api/user/urls.
func ExampleURLHandler_DeleteUserURLs() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{
		OriginalURL: "https://practicum.yandex.ru",
		UserID:      "user-1",
	})

	svc := service.NewURLService(repo, "http://localhost:8080")
	h := handler.NewURLHandler(svc, "http://localhost:8080", nil)

	body := `["abc123"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, "user-1")

	w := httptest.NewRecorder()
	h.DeleteUserURLs(w, req)

	fmt.Println("status:", w.Code)
	// Output:
	// status: 202
}
