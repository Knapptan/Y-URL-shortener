// Package config обеспечивает парсинг конфигурации сервиса через флаги CLI,
// переменные окружения и JSON-файл конфигурации.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// Константы конфигурации: значения по умолчанию и имена переменных окружения.
const (
	// DefaultServerAddress — адрес сервера по умолчанию.
	DefaultServerAddress = "localhost:8080"
	// DefaultBaseURL — базовый URL по умолчанию.
	DefaultBaseURL = "http://localhost:8080"

	// EnvServerAddress — имя переменной окружения для адреса сервера.
	EnvServerAddress = "SERVER_ADDRESS"
	// EnvBaseURL — имя переменной окружения для базового URL.
	EnvBaseURL = "BASE_URL"

	// EnvFileStoragePath — имя переменной окружения для пути к файлу хранилища.
	EnvFileStoragePath = "FILE_STORAGE_PATH"
	// DefaultFileStoragePath — путь к файлу хранилища по умолчанию.
	DefaultFileStoragePath = "storage.json"

	// EnvDatabaseDSN — имя переменной окружения для DSN базы данных.
	EnvDatabaseDSN = "DATABASE_DSN"
	// DefaultDatabaseDSN — DSN базы данных по умолчанию (пустое значение отключает БД).
	DefaultDatabaseDSN = ""

	// EnvSecretKey — имя переменной окружения для секретного ключа подписи cookie.
	EnvSecretKey = "SECRET_KEY"
	// DefaultSecretKey — секретный ключ по умолчанию (только для локальной разработки).
	DefaultSecretKey = "your-secret-key"

	// EnvAuditFile — имя переменной окружения для пути к файлу аудита.
	EnvAuditFile = "AUDIT_FILE"
	// EnvAuditURL — имя переменной окружения для URL удалённого приёмника аудита.
	EnvAuditURL = "AUDIT_URL"

	// EnvEnableHTTPS — имя переменной окружения для включения HTTPS.
	EnvEnableHTTPS = "ENABLE_HTTPS"

	// EnvConfig — имя переменной окружения для пути к JSON-файлу конфигурации.
	EnvConfig = "CONFIG"
)

// Config хранит итоговые настройки сервиса.
type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	SecretKey       string
	AuditFile       string
	AuditURL        string
	EnableHTTPS     bool
}

// fileConfig — структура для чтения JSON-файла конфигурации.
// Теги соответствуют полям из ТЗ инкремента 25.
type fileConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	SecretKey       string `json:"secret_key,omitempty"`
	AuditFile       string `json:"audit_file,omitempty"`
	AuditURL        string `json:"audit_url,omitempty"`
	EnableHTTPS     bool   `json:"enable_https"`
}

// loadFileConfig читает JSON-файл конфигурации.
func loadFileConfig(path string) (*fileConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var cfg fileConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}
	return &cfg, nil
}

// pickString выбирает значение по приоритету:
// env > flag (если задан и отличается от дефолта) > file > flag (если задан) > default.
func pickString(env, flagVal, def, fileVal string) string {
	if env != "" {
		return env
	}
	// Флаг явно задан и не совпадает с дефолтом.
	if flagVal != "" && flagVal != def {
		return flagVal
	}
	// Значение из файла.
	if fileVal != "" {
		return fileVal
	}
	// Флаг задан, но равен дефолту.
	if flagVal != "" {
		return flagVal
	}
	return def
}

// pickBool выбирает булево значение по приоритету:
// env=true > flag=true > file=true > false.
func pickBool(flagVal, envVal, fileVal bool) bool {
	return envVal || flagVal || fileVal
}

// resolveConfig собирает итоговую конфигурацию.
func resolveConfig(
	flagServer, flagBase, envServer, envBase string,
	flagFile, envFile, flagDSN, envDSN string,
	flagSecret, envSecret string,
	flagAuditFile, envAuditFile, flagAuditURL, envAuditURL string,
	flagEnableHTTPS, envEnableHTTPS bool,
	fileCfg *fileConfig,
) *Config {
	fc := fileConfig{}
	if fileCfg != nil {
		fc = *fileCfg
	}

	server := pickString(envServer, flagServer, DefaultServerAddress, fc.ServerAddress)
	base := pickString(envBase, flagBase, DefaultBaseURL, fc.BaseURL)
	filePath := pickString(envFile, flagFile, DefaultFileStoragePath, fc.FileStoragePath)
	dsn := pickString(envDSN, flagDSN, DefaultDatabaseDSN, fc.DatabaseDSN)
	secret := pickString(envSecret, flagSecret, DefaultSecretKey, fc.SecretKey)
	auditFile := pickString(envAuditFile, flagAuditFile, "", fc.AuditFile)
	auditURL := pickString(envAuditURL, flagAuditURL, "", fc.AuditURL)
	enableHTTPS := pickBool(flagEnableHTTPS, envEnableHTTPS, fc.EnableHTTPS)

	return &Config{
		ServerAddress:   server,
		BaseURL:         base,
		FileStoragePath: filePath,
		DatabaseDSN:     dsn,
		SecretKey:       secret,
		AuditFile:       auditFile,
		AuditURL:        auditURL,
		EnableHTTPS:     enableHTTPS,
	}
}

// ParseFlags парсит флаги CLI, переменные окружения и JSON-файл конфигурации.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
		filePath      string
		databaseDSN   string
		secretKey     string
		auditFile     string
		auditURL      string
		configPath    string
		enableHTTPS   bool
	)
	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&filePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "database DSN")
	flag.StringVar(&secretKey, "k", DefaultSecretKey, "secret key for signing cookies")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit remote server URL")
	flag.BoolVar(&enableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&configPath, "c", "", "config file path (JSON)")
	flag.StringVar(&configPath, "config", "", "config file path (JSON)")
	flag.Parse()

	// Определяем путь к конфиг-файлу: флаг > env.
	if configPath == "" {
		configPath = os.Getenv(EnvConfig)
	}

	// Загружаем конфиг-файл, если путь задан.
	var fileCfg *fileConfig
	if configPath != "" {
		loaded, err := loadFileConfig(configPath)
		if err != nil {
			// Не падаем: файл может быть недоступен, продолжаем с флагами/env.
			fmt.Fprintf(os.Stderr, "config: failed to load %q: %v\n", configPath, err)
		} else {
			fileCfg = loaded
		}
	}

	envServer, _ := os.LookupEnv(EnvServerAddress)
	envBase, _ := os.LookupEnv(EnvBaseURL)
	envFile, _ := os.LookupEnv(EnvFileStoragePath)
	envDSN, _ := os.LookupEnv(EnvDatabaseDSN)
	envSecret, _ := os.LookupEnv(EnvSecretKey)
	envAuditFile, _ := os.LookupEnv(EnvAuditFile)
	envAuditURL, _ := os.LookupEnv(EnvAuditURL)
	envHTTPS := os.Getenv(EnvEnableHTTPS) == "true" || os.Getenv(EnvEnableHTTPS) == "1"

	return resolveConfig(
		serverAddress, baseURL, envServer, envBase,
		filePath, envFile, databaseDSN, envDSN,
		secretKey, envSecret,
		auditFile, envAuditFile, auditURL, envAuditURL,
		enableHTTPS, envHTTPS,
		fileCfg,
	)
}
