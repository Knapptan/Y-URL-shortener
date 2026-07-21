package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

// mockRepo — реализация интерфейса repository.URLRepository для тестов
type mockRepo struct {
	saveFunc func(string) (string, error)
	getFunc  func(string) (string, bool)
}

func (m *mockRepo) Save(originalURL string) (string, error) {
	return m.saveFunc(originalURL)
}

func (m *mockRepo) Get(id string) (string, bool) {
	return m.getFunc(id)
}

func TestHandler_CreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSave       func(string) (string, error)
		expectedStatus int
		expectedBody   string // проверяем частично, например, содержит ли http://localhost:8080/
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string) (string, error) {
				return "abc123", nil
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "http://localhost:8080/abc123",
		},
		{
			name:           "empty body",
			method:         http.MethodPost,
			body:           "",
			mockSave:       nil, // не будет вызван
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty body\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string) (string, error) {
				return "", errors.New("some error")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some error\n",
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           "",
			mockSave:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Only POST allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Собираем зависимости с моком
			repo := &mockRepo{
				saveFunc: tt.mockSave,
			}
			svc := service.NewURLService(repo)
			h := NewURLHandler(svc)

			// Создаём запрос
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			// Вызываем хендлер
			h.CreateShortURL(w, req)

			// Проверяем статус
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Проверяем тело (если не пустое ожидание)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_RedirectToOriginal(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		mockGet        func(string) (string, bool)
		expectedStatus int
		expectedHeader map[string]string // для проверки Location
	}{
		{
			name:   "success redirect",
			method: http.MethodGet,
			path:   "/abc123",
			mockGet: func(id string) (string, bool) {
				if id == "abc123" {
					return "https://ya.ru", true
				}
				return "", false
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: map[string]string{"Location": "https://ya.ru"},
		},
		{
			name:           "missing id",
			method:         http.MethodGet,
			path:           "/",
			mockGet:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/notfound",
			mockGet: func(_ string) (string, bool) {
				return "", false
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "method not allowed",
			method:         http.MethodPost,
			path:           "/abc123",
			mockGet:        nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				getFunc: tt.mockGet,
			}
			svc := service.NewURLService(repo)
			h := NewURLHandler(svc)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			h.RedirectToOriginal(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			for key, val := range tt.expectedHeader {
				assert.Equal(t, val, w.Header().Get(key))
			}
		})
	}
}
