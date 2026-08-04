package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	// Тестовый обработчик, который возвращает JSON
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"ok"}`))
	})

	// Тест: запрос без сжатия, клиент поддерживает gzip -> ответ должен быть сжат
	t.Run("request_no_gzip_response_compressed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		handler := GzipMiddleware(testHandler)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

		// Распаковываем тело
		reader, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, `{"message":"ok"}`, string(body))
	})

	// Тест: клиент не поддерживает gzip -> ответ без сжатия
	t.Run("client_no_gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// не ставим Accept-Encoding
		w := httptest.NewRecorder()

		handler := GzipMiddleware(testHandler)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Content-Encoding"))
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"message":"ok"}`, string(body))
	})

	// Тест: запрос сжатый, клиент поддерживает gzip – работает и запрос, и ответ
	t.Run("compressed_request_and_response", func(t *testing.T) {
		// Создаём сжатое тело запроса
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(`{"key":"value"}`))
		require.NoError(t, err)
		gz.Close()

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Обработчик, который читает тело и возвращает его же
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		})
		mw := GzipMiddleware(handler)
		mw.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

		// Распаковываем ответ
		gr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		bodyBytes, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.Equal(t, `{"key":"value"}`, string(bodyBytes))
	})

	// Тест: ответ с text/html должен сжиматься
	t.Run("html_content_compressed", func(t *testing.T) {
		htmlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><body>Hello</body></html>"))
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		mw := GzipMiddleware(htmlHandler)
		mw.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		body, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Hello")
	})

	// Тест: ответ с text/plain не должен сжиматься
	t.Run("plain_text_not_compressed", func(t *testing.T) {
		plainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("plain text"))
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		mw := GzipMiddleware(plainHandler)
		mw.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Empty(t, resp.Header.Get("Content-Encoding"))
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "plain text", string(body))
	})
}
