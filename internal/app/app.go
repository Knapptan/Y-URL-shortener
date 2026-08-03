package app

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// Run запускает HTTP-сервер с заданной конфигурацией.
func Run(cfg *config.Config) error {
	// Настройка логгера
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Infof("Starting server on %s", cfg.ServerAddress)
	logrus.SetOutput(os.Stdout)
	// Инициализация зависимостей
	repo := repository.NewInMemoryRepo()
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)

	r := chi.NewRouter()

	// Подключаем кастомный middleware логирования (на logrus)
	r.Use(middleware.LoggingMiddleware)

	// Регистрируем маршруты
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)

	// Запуск сервера
	return http.ListenAndServe(cfg.ServerAddress, r)
}
