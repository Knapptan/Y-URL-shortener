package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	// Отключаем вывод логов для чистоты тестов
	logrus.SetLevel(logrus.PanicLevel)

	tests := []struct {
		name           string
		handlerStatus  int
		handlerBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "OK response",
			handlerStatus:  http.StatusOK,
			handlerBody:    "ok",
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "created",
			handlerStatus:  http.StatusCreated,
			handlerBody:    "created",
			expectedStatus: http.StatusCreated,
			expectedBody:   "created",
		},
		{
			name:           "bad request",
			handlerStatus:  http.StatusBadRequest,
			handlerBody:    "bad",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "bad",
		},
		{
			name:           "redirect",
			handlerStatus:  http.StatusTemporaryRedirect,
			handlerBody:    "",
			expectedStatus: http.StatusTemporaryRedirect,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый обработчик
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				if tt.handlerBody != "" {
					w.Write([]byte(tt.handlerBody))
				}
			})

			// Оборачиваем middleware
			handler := LoggingMiddleware(next)

			// Выполняем запрос
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Проверяем, что ответ не изменён
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}
