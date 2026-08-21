package config

import (
	"flag"
	"os"
)

const (
	DefaultServerAddress = "localhost:8080"
	DefaultBaseURL       = "http://localhost:8080"

	EnvServerAddress = "SERVER_ADDRESS"
	EnvBaseURL       = "BASE_URL"

	EnvFileStoragePath     = "FILE_STORAGE_PATH"
	DefaultFileStoragePath = "storage.json"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

// BuildConfig определяет конфигурацию на основе значений из флагов и переменных окружения.
// Приоритет: переменная окружения > флаг > значение по умолчанию.
func BuildConfig(flagServerAddress, flagBaseURL string) *Config {
	return &Config{
		ServerAddress: DefaultServerAddress,
		BaseURL:       DefaultBaseURL,
	}
}

// resolveConfig определяет итоговые значения с учётом приоритета:
// переменная окружения > переданный флаг > значение по умолчанию.
func resolveConfig(flagServer, flagBase, envServer, envBase, flagFile, envFile string) *Config {
	server := flagServer
	if envServer != "" {
		server = envServer
	}
	base := flagBase
	if envBase != "" {
		base = envBase
	}
	filePath := flagFile
	if envFile != "" {
		filePath = envFile
	}
	if filePath == "" {
		filePath = DefaultFileStoragePath
	}
	return &Config{
		ServerAddress:   server,
		BaseURL:         base,
		FileStoragePath: filePath,
	}
}

// ParseFlags парсит флаги командной строки и переменные окружения,
// возвращает конфигурацию с учётом приоритета.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
		filePath      string
	)
	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&filePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.Parse()

	envServer := os.Getenv(EnvServerAddress)
	envBase := os.Getenv(EnvBaseURL)
	envFile := os.Getenv(EnvFileStoragePath)

	return resolveConfig(serverAddress, baseURL, envServer, envBase, filePath, envFile)
}
