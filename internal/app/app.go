package app

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// Run запускает HTTP-сервер с заданной конфигурацией.
func Run(cfg *config.Config) error {
	// Настройка логгера
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// Создаём репозиторий с файловым хранилищем
	repo, err := repository.NewFileRepository(cfg.FileStoragePath)
	if err != nil {
		slog.Error("Failed to init storage", "error", err)
		return err // Если repo не удалось проинициализировать
	}
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)

	r := chi.NewRouter()

	// Подключаем middleware
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	// Регистрируем маршруты
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)
	r.Post("/api/shorten", h.CreateShortenJSON)

	// Запуск сервера
	return http.ListenAndServe(cfg.ServerAddress, r)
}
