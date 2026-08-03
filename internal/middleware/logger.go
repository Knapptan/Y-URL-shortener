package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// responseWriter обёртка над http.ResponseWriter для захвата статуса и размера ответа.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK // если WriteHeader не вызывали, статус 200
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// LoggingMiddleware логирует запросы и ответы.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.Info("Middleware called for request")
		start := time.Now()

		// Оборачиваем ResponseWriter
		rw := &responseWriter{
			ResponseWriter: w,
			status:         0,
			size:           0,
		}

		// Вызываем следующий обработчик
		next.ServeHTTP(rw, r)

		// Логируем информацию
		duration := time.Since(start)
		logrus.WithFields(logrus.Fields{
			"method":   r.Method,
			"uri":      r.URL.Path,
			"duration": duration,
			"status":   rw.status,
			"size":     rw.size,
		}).Info("HTTP request")
	})
}
