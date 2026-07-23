package main

import (
	"log"

	"github.com/Knapptan/Y-URL-shortener/internal/app"
	"github.com/Knapptan/Y-URL-shortener/internal/config"
)

func main() {
	cfg := config.ParseFlags()
	if err := app.Run(cfg); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
