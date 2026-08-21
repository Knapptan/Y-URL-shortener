package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

// mockRepo реализует repository.URLRepository для тестов
type mockRepo struct {
	saveFunc      func(string, model.URLRecord) error
	getFunc       func(string) (model.URLRecord, bool)
	saveBatchFunc func(map[string]model.URLRecord) error
}

func (m *mockRepo) Save(id string, record model.URLRecord) error {
	return m.saveFunc(id, record)
}

func (m *mockRepo) Get(id string) (model.URLRecord, bool) {
	return m.getFunc(id)
}

func (m *mockRepo) SaveBatch(batch map[string]model.URLRecord) error {
	return m.saveBatchFunc(batch)
}

func TestHandler_CreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSave       func(string, model.URLRecord) error
		expectedStatus int
		expectedPrefix string
		expectedBody   string
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.DefaultBaseURL + "/",
		},
		{
			name:           "empty body",
			method:         http.MethodPost,
			body:           "",
			mockSave:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty body\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string, _ model.URLRecord) error {
				return errors.New("some error")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveFunc: tt.mockSave,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			h.CreateShortURL(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedPrefix != "" {
				body := w.Body.String()
				assert.True(t, strings.HasPrefix(body, tt.expectedPrefix), "ответ должен начинаться с %s, получено %s", tt.expectedPrefix, body)
				id := strings.TrimPrefix(body, tt.expectedPrefix)
				assert.NotEmpty(t, id, "ID не должен быть пустым")
			}
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
		mockGet        func(string) (model.URLRecord, bool)
		expectedStatus int
		expectedHeader map[string]string
	}{
		{
			name:   "success redirect",
			method: http.MethodGet,
			path:   "/abc123",
			mockGet: func(id string) (model.URLRecord, bool) {
				if id == "abc123" {
					return model.URLRecord{OriginalURL: "https://ya.ru"}, true
				}
				return model.URLRecord{}, false
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
			mockGet: func(_ string) (model.URLRecord, bool) {
				return model.URLRecord{}, false
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				getFunc: tt.mockGet,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

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

func TestHandler_CreateShortenJSON(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSave       func(string, model.URLRecord) error
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   `{"url":"https://ya.ru"}`,
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			checkJSON:      true,
		},
		{
			name:           "empty url",
			method:         http.MethodPost,
			body:           `{"url":""}`,
			mockSave:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "URL is empty\n",
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `{"url": "missing quote}`,
			mockSave:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body:   `{"url":"https://ya.ru"}`,
			mockSave: func(_ string, _ model.URLRecord) error {
				return errors.New("some repo error")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some repo error\n",
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           "",
			mockSave:       nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Only POST allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveFunc: tt.mockSave,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateShortenJSON(w, req)

			if tt.checkJSON {
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				result, ok := resp["result"]
				assert.True(t, ok, "поле result отсутствует")
				assert.True(t, strings.HasPrefix(result, config.DefaultBaseURL+"/"), "неправильный формат URL")
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				if strings.HasPrefix(tt.expectedBody, "{") {
					assert.JSONEq(t, tt.expectedBody, w.Body.String())
				} else {
					assert.Equal(t, tt.expectedBody, w.Body.String())
				}
			}
		})
	}
}

func TestHandler_CreateShortenBatch(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSaveBatch  func(map[string]model.URLRecord) error
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:   "success batch",
			method: http.MethodPost,
			body: `[
				{"correlation_id":"1", "original_url":"https://ya.ru"},
				{"correlation_id":"2", "original_url":"https://google.com"}
			]`,
			mockSaveBatch: func(_ map[string]model.URLRecord) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			checkJSON:      true,
		},
		{
			name:           "empty batch",
			method:         http.MethodPost,
			body:           `[]`,
			mockSaveBatch:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty batch\n",
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `[{"correlation_id":"1", "original_url":}`,
			mockSaveBatch:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body: `[
				{"correlation_id":"1", "original_url":"https://ya.ru"}
			]`,
			mockSaveBatch: func(_ map[string]model.URLRecord) error {
				return errors.New("some repo error")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some repo error\n",
		},
		{
			name:           "method not allowed",
			method:         http.MethodGet,
			body:           "",
			mockSaveBatch:  nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Only POST allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveBatchFunc: tt.mockSaveBatch,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/api/shorten/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateShortenBatch(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkJSON {
				var resp []map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp, 2)
				for _, item := range resp {
					assert.Contains(t, item, "correlation_id")
					assert.Contains(t, item, "short_url")
					assert.True(t, strings.HasPrefix(item["short_url"], config.DefaultBaseURL+"/"))
				}
			}
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}
