package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

func main() {
	// инициализируем слои
	repo := repository.NewInMemoryRepo()
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc)

	// Создаём роутер
	r := chi.NewRouter()

	// Регистрируем маршруты
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)

	// Запускаем сервер на порту 8080
	println("Server is running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}
