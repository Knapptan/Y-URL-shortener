package app

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"

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

	// Инициализация БД (если DSN указан)
	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		var err error
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			slog.Error("Failed to open database", "error", err)
			return err
		}
		if err := db.Ping(); err != nil {
			slog.Error("Failed to ping database", "error", err)
			return err
		}
		slog.Info("Database connected")
	} else {
		slog.Info("Database DSN not provided, running without DB")
	}
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// Репозиторий (пока файловый)
	repo, err := repository.NewFileRepository(cfg.FileStoragePath)
	if err != nil {
		slog.Error("Failed to init storage", "error", err)
		return err
	}
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)

	// Хендлер для ping
	pingHandler := handler.NewPingHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	// Регистрация маршрутов
	r.Get("/ping", pingHandler.Ping)
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)
	r.Post("/api/shorten", h.CreateShortenJSON)

	slog.Info("Starting server", "address", cfg.ServerAddress)
	return http.ListenAndServe(cfg.ServerAddress, r)
}
