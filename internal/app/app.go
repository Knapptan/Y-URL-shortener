package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// Run запускает HTTP-сервер с заданной конфигурацией.
func Run(cfg *config.Config) error {
	// Инициализация зависимостей
	repo := repository.NewInMemoryRepo()
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)

	// Настройка роутера
	r := chi.NewRouter()
	r.Use(middleware.Logger) // можно добавить логирование
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)

	// Запуск сервера
	return http.ListenAndServe(cfg.ServerAddress, r)
}
