package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name              string
		flagServer        string
		flagBase          string
		envServer         string
		envBase           string
		flagFile          string
		envFile           string
		flagDSN           string
		envDSN            string
		flagSecret        string
		envSecret         string
		flagAuditFile     string
		envAuditFile      string
		flagAuditURL      string
		envAuditURL       string
		expectedServer    string
		expectedBase      string
		expectedFile      string
		expectedDSN       string
		expectedSecret    string
		expectedAuditFile string
		expectedAuditURL  string
	}{
		{
			name:           "no flags, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			flagFile:       "",
			flagDSN:        "",
			flagSecret:     DefaultSecretKey,
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "",
			expectedSecret: DefaultSecretKey,
		},
		{
			name:           "flag only",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			flagFile:       "",
			flagDSN:        "postgres://user:pass@localhost/db",
			flagSecret:     "flag-secret",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:8888",
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://user:pass@localhost/db",
			expectedSecret: "flag-secret",
		},
		{
			name:           "env wins",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			flagSecret:     "flag-secret",
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			envSecret:      "env-secret",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
			expectedSecret: "env-secret",
		},
		{
			name:              "audit flags only",
			flagServer:        DefaultServerAddress,
			flagBase:          DefaultBaseURL,
			flagSecret:        DefaultSecretKey,
			flagAuditFile:     "/tmp/audit.log",
			flagAuditURL:      "http://audit.local/events",
			expectedServer:    DefaultServerAddress,
			expectedBase:      DefaultBaseURL,
			expectedFile:      DefaultFileStoragePath,
			expectedSecret:    DefaultSecretKey,
			expectedAuditFile: "/tmp/audit.log",
			expectedAuditURL:  "http://audit.local/events",
		},
		{
			name:              "audit env wins over flags",
			flagServer:        DefaultServerAddress,
			flagBase:          DefaultBaseURL,
			flagSecret:        DefaultSecretKey,
			flagAuditFile:     "/tmp/flag-audit.log",
			envAuditFile:      "/tmp/env-audit.log",
			flagAuditURL:      "http://flag-audit.local/events",
			envAuditURL:       "http://env-audit.local/events",
			expectedServer:    DefaultServerAddress,
			expectedBase:      DefaultBaseURL,
			expectedFile:      DefaultFileStoragePath,
			expectedSecret:    DefaultSecretKey,
			expectedAuditFile: "/tmp/env-audit.log",
			expectedAuditURL:  "http://env-audit.local/events",
		},
		{
			name:              "audit disabled by default",
			flagServer:        DefaultServerAddress,
			flagBase:          DefaultBaseURL,
			flagSecret:        DefaultSecretKey,
			expectedServer:    DefaultServerAddress,
			expectedBase:      DefaultBaseURL,
			expectedFile:      DefaultFileStoragePath,
			expectedSecret:    DefaultSecretKey,
			expectedAuditFile: "",
			expectedAuditURL:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveConfig(
				tt.flagServer, tt.flagBase, tt.envServer, tt.envBase,
				tt.flagFile, tt.envFile, tt.flagDSN, tt.envDSN,
				tt.flagSecret, tt.envSecret, tt.flagAuditFile, tt.envAuditFile,
				tt.flagAuditURL, tt.envAuditURL,
			)
			assert.Equal(t, tt.expectedServer, cfg.ServerAddress)
			assert.Equal(t, tt.expectedBase, cfg.BaseURL)
			assert.Equal(t, tt.expectedFile, cfg.FileStoragePath)
			assert.Equal(t, tt.expectedDSN, cfg.DatabaseDSN)
			assert.Equal(t, tt.expectedSecret, cfg.SecretKey)
			assert.Equal(t, tt.expectedAuditFile, cfg.AuditFile)
			assert.Equal(t, tt.expectedAuditURL, cfg.AuditURL)
		})
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("flag overrides default", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-s", "my-secret"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		t.Setenv(EnvSecretKey, "")
		t.Setenv(EnvAuditFile, "")
		t.Setenv(EnvAuditURL, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:8888", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "my-secret", cfg.SecretKey)
	})

	t.Run("env overrides flag", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-s", "flag-secret"}
		t.Setenv(EnvServerAddress, "localhost:9999")
		t.Setenv(EnvBaseURL, "http://localhost:9999")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		t.Setenv(EnvSecretKey, "env-secret")
		t.Setenv(EnvAuditFile, "")
		t.Setenv(EnvAuditURL, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:9999", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:9999", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "env-secret", cfg.SecretKey)
	})

	t.Run("audit flags and env", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-audit-file", "/tmp/flag-audit.log", "-audit-url", "http://flag-audit.local/events"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		t.Setenv(EnvSecretKey, "")
		t.Setenv(EnvAuditFile, "")
		t.Setenv(EnvAuditURL, "")
		cfg := ParseFlags()
		assert.Equal(t, "/tmp/flag-audit.log", cfg.AuditFile)
		assert.Equal(t, "http://flag-audit.local/events", cfg.AuditURL)

		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-audit-file", "/tmp/flag-audit.log", "-audit-url", "http://flag-audit.local/events"}
		t.Setenv(EnvAuditFile, "/tmp/env-audit.log")
		t.Setenv(EnvAuditURL, "http://env-audit.local/events")
		cfg = ParseFlags()
		assert.Equal(t, "/tmp/env-audit.log", cfg.AuditFile)
		assert.Equal(t, "http://env-audit.local/events", cfg.AuditURL)
	})
}

func TestParseFlagsDefaults(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	t.Setenv(EnvServerAddress, "")
	t.Setenv(EnvBaseURL, "")
	t.Setenv(EnvFileStoragePath, "")
	t.Setenv(EnvDatabaseDSN, "")
	t.Setenv(EnvSecretKey, "")

	cfg := ParseFlags()
	assert.Equal(t, DefaultServerAddress, cfg.ServerAddress)
	assert.Equal(t, DefaultBaseURL, cfg.BaseURL)
	assert.Equal(t, DefaultSecretKey, cfg.SecretKey)
}
