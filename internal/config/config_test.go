package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name           string
		flagServer     string
		flagBase       string
		envServer      string
		envBase        string
		flagFile       string
		envFile        string
		expectedServer string
		expectedBase   string
		expectedFile   string
	}{
		{
			name:           "no flags, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
		},
		{
			name:           "flag only",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:8888",
			expectedFile:   DefaultFileStoragePath,
		},
		{
			name:           "env only",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			flagFile:       "",
			envFile:        "",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
		},
		{
			name:           "flag and env – env wins",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			flagFile:       "",
			envFile:        "",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
		},
		{
			name:           "env only for server, flag only for base",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "localhost:7777",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			expectedServer: "localhost:7777",
			expectedBase:   "http://localhost:8888",
			expectedFile:   DefaultFileStoragePath,
		},
		{
			name:           "empty env values ignored",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "",
			envBase:        "http://localhost:9999",
			flagFile:       "",
			envFile:        "",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
		},
		// Можно добавить тесты для файлового пути, но для этого инкремента не обязательно
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveConfig(tt.flagServer, tt.flagBase, tt.envServer, tt.envBase, tt.flagFile, tt.envFile)
			assert.Equal(t, tt.expectedServer, cfg.ServerAddress)
			assert.Equal(t, tt.expectedBase, cfg.BaseURL)
			assert.Equal(t, tt.expectedFile, cfg.FileStoragePath)
		})
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("flag overrides default", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvFileStoragePath, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:8888", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
	})

	t.Run("env overrides flag", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888"}
		t.Setenv(EnvServerAddress, "localhost:9999")
		t.Setenv(EnvBaseURL, "http://localhost:9999")
		t.Setenv(EnvFileStoragePath, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:9999", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:9999", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
	})
}
