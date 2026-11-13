package main

import (
	"log"
	"log/slog"

	"github.com/Knapptan/Y-URL-shortener/internal/app"
	"github.com/Knapptan/Y-URL-shortener/internal/config"
)

func main() {
	cfg := config.ParseFlags()
	if err := app.Run(cfg); err != nil {
		slog.Error("Server failed", "error", err)
		log.Fatal(err)
	}
}
