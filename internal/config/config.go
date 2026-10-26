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

	EnvDatabaseDSN     = "DATABASE_DSN"
	DefaultDatabaseDSN = ""

	EnvSecretKey     = "SECRET_KEY"
	DefaultSecretKey = "your-secret-key"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	SecretKey       string
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
func resolveConfig(flagServer, flagBase, envServer, envBase, flagFile, envFile, flagDSN, envDSN, flagSecret, envSecret string) *Config {
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
	dsn := flagDSN
	if envDSN != "" {
		dsn = envDSN
	}
	secret := flagSecret
	if envSecret != "" {
		secret = envSecret
	}
	if secret == "" {
		secret = DefaultSecretKey
	}
	// если dsn == "" – оставляем пустым
	return &Config{
		ServerAddress:   server,
		BaseURL:         base,
		FileStoragePath: filePath,
		DatabaseDSN:     dsn,
		SecretKey:       secret,
	}
}

// ParseFlags парсит флаги командной строки и переменные окружения,
// возвращает конфигурацию с учётом приоритета.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
		filePath      string
		databaseDSN   string
		secretKey     string
	)
	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&filePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "database DSN")
	flag.StringVar(&secretKey, "s", DefaultSecretKey, "secret key for signing cookies")
	flag.Parse()

	envServer := os.Getenv(EnvServerAddress)
	envBase := os.Getenv(EnvBaseURL)
	envFile := os.Getenv(EnvFileStoragePath)
	envDSN := os.Getenv(EnvDatabaseDSN)
	envSecret := os.Getenv(EnvSecretKey)

	return resolveConfig(serverAddress, baseURL, envServer, envBase, filePath, envFile, databaseDSN, envDSN, secretKey, envSecret)
}
