package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

func main() {
	// Парсим флаги
	cfg := config.ParseFlags()

	// Инициализация зависимостей
	repo := repository.NewInMemoryRepo()
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)

	// Настройка роутера
	r := chi.NewRouter()
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)

	// Запуск сервера на адресе из флага -a
	println("Server is running on", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		panic(err)
	}
}
