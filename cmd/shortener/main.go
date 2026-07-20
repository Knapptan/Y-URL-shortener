package main

import (
	"net/http"

	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

func main() {
	// инициализируем слои
	repo := repository.NewInMemoryRepo()
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc)

	// регистрируем маршруты
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// если путь = "/" и метод POST -> создание
		// если путь содержит ID (т.е. не просто "/") -> редирект
		if r.URL.Path == "/" {
			h.CreateShortURL(w, r)
		} else {
			h.RedirectToOriginal(w, r)
		}
	})

	// запускаем сервер на порту 8080
	println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
