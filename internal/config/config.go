// Package config обеспечивает парсинг конфигурации сервиса через флаги CLI,
// переменные окружения и JSON-файл конфигурации.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
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

	// EnvTrustedSubnet — имя переменной окружения для доверенной подсети (CIDR).
	EnvTrustedSubnet = "TRUSTED_SUBNET"
	// DefaultTrustedSubnet — пустое значение означает «доступ запрещён всем».
	DefaultTrustedSubnet = ""
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
	TrustedSubnet   string
	EnableHTTPS     bool
}

// fileConfig — JSON-представление конфигурации из -c.
type fileConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	SecretKey       string `json:"secret_key"`
	AuditFile       string `json:"audit_file"`
	AuditURL        string `json:"audit_url"`
	TrustedSubnet   string `json:"trusted_subnet"`
	EnableHTTPS     bool   `json:"enable_https"`
}

// loadFileConfig читает JSON-файл конфигурации.
func loadFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &fc, nil
}

// pick выбирает значение по приоритету:
// env (если задан) > flag (если явно передан в CLI) > file > default.
//
// Ключевое отличие от наивной реализации: флаг считается «заданным»,
// если его имя вернул flag.Visit — то есть пользователь действительно
// передал его в командной строке. Это позволяет отличить
// «-a localhost:8080 передали явно» от «-a не передавали,
// значение совпало с дефолтом».
func pick(env, flagVal string, flagSet bool, fileVal, def string) string {
	if env != "" {
		return env
	}
	if flagSet {
		return flagVal
	}
	if fileVal != "" {
		return fileVal
	}
	return def
}

// pickBool — булев аналог pick. envSet/flagSet показывают, было ли
// значение задано явно (в env или CLI). Это позволяет отличить
// «ENABLE_HTTPS не задан» от «ENABLE_HTTPS=false».
func pickBool(envVal, envSet, flagVal, flagSet, fileVal bool) bool {
	if envSet {
		return envVal
	}
	if flagSet {
		return flagVal
	}
	return fileVal
}

// envBool читает переменную окружения как bool.
// Возвращает (значение, было ли переменная задана).
func envBool(key string) (bool, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}
	return b, true
}

// ParseFlags парсит флаги, читает env и файл, собирает конфиг
// с приоритетом: env > flag (явно задан) > file > default.
func ParseFlags() *Config {
	var (
		serverAddress   string
		baseURL         string
		fileStoragePath string
		databaseDSN     string
		secretKey       string
		auditFile       string
		auditURL        string
		configPath      string
		trustedSubnet   string
		enableHTTPS     bool
	)

	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&fileStoragePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "database DSN")
	flag.StringVar(&secretKey, "s", DefaultSecretKey, "secret key for signing cookies")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit remote server URL")
	flag.StringVar(&configPath, "c", "", "path to config file")
	flag.StringVar(&trustedSubnet, "t", DefaultTrustedSubnet, "trusted subnet in CIDR notation")
	flag.BoolVar(&enableHTTPS, "https", false, "enable HTTPS")
	flag.Parse()

	// Собираем множество имён флагов, реально переданных в CLI.
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Путь к файлу тоже может прийти из env (EnvConfig), если флаг не задан.
	configPath = pick(os.Getenv(EnvConfig), configPath, setFlags["c"], "", "")

	// Читаем файл, если путь известен.
	var fileCfg fileConfig
	if configPath != "" {
		fc, err := loadFileConfig(configPath)
		if err != nil {
			log.Fatalf("load config file %q: %v", configPath, err)
		}
		fileCfg = *fc
	}

	envHTTPSVal, envHTTPSSet := envBool(EnvEnableHTTPS)

	return &Config{
		ServerAddress:   pick(os.Getenv(EnvServerAddress), serverAddress, setFlags["a"], fileCfg.ServerAddress, DefaultServerAddress),
		BaseURL:         pick(os.Getenv(EnvBaseURL), baseURL, setFlags["b"], fileCfg.BaseURL, DefaultBaseURL),
		FileStoragePath: pick(os.Getenv(EnvFileStoragePath), fileStoragePath, setFlags["f"], fileCfg.FileStoragePath, DefaultFileStoragePath),
		DatabaseDSN:     pick(os.Getenv(EnvDatabaseDSN), databaseDSN, setFlags["d"], fileCfg.DatabaseDSN, DefaultDatabaseDSN),
		SecretKey:       pick(os.Getenv(EnvSecretKey), secretKey, setFlags["s"], fileCfg.SecretKey, DefaultSecretKey),
		AuditFile:       pick(os.Getenv(EnvAuditFile), auditFile, setFlags["audit-file"], fileCfg.AuditFile, ""),
		AuditURL:        pick(os.Getenv(EnvAuditURL), auditURL, setFlags["audit-url"], fileCfg.AuditURL, ""),
		TrustedSubnet:   pick(os.Getenv(EnvTrustedSubnet), trustedSubnet, setFlags["t"], fileCfg.TrustedSubnet, DefaultTrustedSubnet),
		EnableHTTPS:     pickBool(envHTTPSVal, envHTTPSSet, enableHTTPS, setFlags["https"], fileCfg.EnableHTTPS),
	}
}
