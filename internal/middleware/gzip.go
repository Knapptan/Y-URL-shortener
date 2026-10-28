package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

func getGzipWriter(w io.Writer) *gzip.Writer {
	gz := gzipWriterPool.Get().(*gzip.Writer)
	gz.Reset(w)
	return gz
}

func putGzipWriter(gz *gzip.Writer) {
	gz.Close()
	gzipWriterPool.Put(gz)
}

func shouldCompress(contentType string) bool {
	if contentType == "" {
		return false
	}
	lower := strings.ToLower(contentType)
	return strings.HasPrefix(lower, "application/json") ||
		strings.HasPrefix(lower, "text/html")
}

// GzipMiddleware реализует поддержку gzip-сжатия запросов и ответов.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			r.Body = gzReader
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gzw := &gzipResponseWriter{ResponseWriter: w, origWriter: w}
		defer gzw.Close()
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	origWriter  io.Writer
	gz          *gzip.Writer
	compress    bool
	wroteHeader bool
}

// WriteHeader перехватывает статус и при необходимости включает сжатие.
func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	if shouldCompress(g.Header().Get("Content-Type")) {
		g.compress = true
		g.gz = getGzipWriter(g.origWriter)
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
	}
	g.ResponseWriter.WriteHeader(statusCode)
}

// Write передаёт данные в gzip.Writer или в оригинальный writer.
func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.compress && g.gz != nil {
		return g.gz.Write(b)
	}
	return g.origWriter.Write(b)
}

// Close возвращает gzip.Writer в пул.
func (g *gzipResponseWriter) Close() error {
	if g.gz != nil {
		putGzipWriter(g.gz)
	}
	return nil
}
