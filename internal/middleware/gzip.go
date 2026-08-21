package middleware

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// shouldCompress проверяет, нужно ли сжимать ответ для данного Content-Type.
func shouldCompress(contentType string) bool {
	if contentType == "" {
		return false
	}
	lower := strings.ToLower(contentType)
	return strings.HasPrefix(lower, "application/json") ||
		strings.HasPrefix(lower, "text/html")
}

// GzipMiddleware реализует middleware для поддержки сжатия gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Распаковка тела запроса
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				slog.Error("Failed to create gzip reader", "error", err)
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}

		// Если клиент не поддерживает gzip – пропускаем без сжатия
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Создаём обёртку, но gzip.Writer пока не создаём
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			origWriter:     w,
		}
		defer gzw.Close()

		next.ServeHTTP(gzw, r)
	})
}

// gzipResponseWriter обёртка над http.ResponseWriter,
// которая сжимает ответ только для разрешённых Content-Type.
type gzipResponseWriter struct {
	http.ResponseWriter
	origWriter  io.Writer    // оригинальный writer для несжатых данных
	gz          *gzip.Writer // создаётся только если нужно сжатие
	compress    bool
	wroteHeader bool
}

// WriteHeader определяет, нужно ли сжимать, и при необходимости создаёт gzip.Writer.
func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true

	ct := g.Header().Get("Content-Type")
	if shouldCompress(ct) {
		g.compress = true
		g.gz = gzip.NewWriter(g.origWriter)
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
	} else {
		g.compress = false
		// gz остаётся nil
	}
	g.ResponseWriter.WriteHeader(statusCode)
}

// Write отправляет данные либо в gzip.Writer, либо напрямую в оригинальный writer.
func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.compress && g.gz != nil {
		return g.gz.Write(b)
	}
	return g.origWriter.Write(b)
}

// Close закрывает gzip.Writer, если он был создан.
func (g *gzipResponseWriter) Close() error {
	if g.gz != nil {
		return g.gz.Close()
	}
	return nil
}
