package main

import (
	"fmt"
	"log"

	"github.com/Knapptan/Y-URL-shortener/internal/app"
	"github.com/Knapptan/Y-URL-shortener/internal/config"
)

// Переменные для инъекции информации о сборке через ldflags:
//
//	go build -ldflags "\
//	  -X main.buildVersion=1.0.0 \
//	  -X main.buildDate=2024-01-15 \
//	  -X main.buildCommit=abc1234" \
//	  -o shortener ./cmd/shortener
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// main запускает сервис.
func main() {
	printBuildInfo()

	cfg := config.ParseFlags()
	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

// printBuildInfo выводит версию, дату сборки и коммит в stdout.
// Если значение не передано через ldflags, выводится "N/A".
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date: %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit: %s\n", valueOrNA(buildCommit))
}

// valueOrNA возвращает "N/A", если строка пустая.
func valueOrNA(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
