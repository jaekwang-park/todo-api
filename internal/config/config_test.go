package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/config"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "DB_SSLMODE", "APP_ENV", "AUTH_DEV_MODE", "LOG_LEVEL",
		"COGNITO_REGION", "COGNITO_USER_POOL_ID", "COGNITO_APP_CLIENT_ID", "COGNITO_APP_CLIENT_SECRET",
	} {
		t.Setenv(key, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg := config.Load()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ServerPort", cfg.ServerPort, "8080"},
		{"AppEnv", cfg.AppEnv, "local"},
		{"DB.Host", cfg.DB.Host, "localhost"},
		{"DB.Port", cfg.DB.Port, "5432"},
		{"DB.User", cfg.DB.User, "todo"},
		{"DB.Password", cfg.DB.Password, "todo"},
		{"DB.Name", cfg.DB.Name, "todo"},
		{"DB.SSLMode", cfg.DB.SSLMode, "disable"},
		{"Cognito.Region", cfg.Cognito.Region, "ap-northeast-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %s, want %s", tt.got, tt.want)
			}
		})
	}

	t.Run("AuthDevMode", func(t *testing.T) {
		if cfg.AuthDevMode {
			t.Errorf("got AuthDevMode=true, want false")
		}
	})

	t.Run("LogLevel", func(t *testing.T) {
		if cfg.LogLevel != "info" {
			t.Errorf("got LogLevel=%s, want info", cfg.LogLevel)
		}
	})
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("APP_ENV", "alpha")
	t.Setenv("AUTH_DEV_MODE", "false")
	t.Setenv("COGNITO_REGION", "us-east-1")
	t.Setenv("COGNITO_USER_POOL_ID", "pool-123")
	t.Setenv("COGNITO_APP_CLIENT_ID", "client-456")
	t.Setenv("COGNITO_APP_CLIENT_SECRET", "secret-789")
	t.Setenv("LOG_LEVEL", "debug")

	cfg := config.Load()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ServerPort", cfg.ServerPort, "9090"},
		{"DB.Host", cfg.DB.Host, "db.example.com"},
		{"DB.Port", cfg.DB.Port, "5433"},
		{"DB.User", cfg.DB.User, "admin"},
		{"DB.Password", cfg.DB.Password, "secret"},
		{"DB.Name", cfg.DB.Name, "mydb"},
		{"DB.SSLMode", cfg.DB.SSLMode, "require"},
		{"AppEnv", cfg.AppEnv, "alpha"},
		{"Cognito.Region", cfg.Cognito.Region, "us-east-1"},
		{"Cognito.UserPoolID", cfg.Cognito.UserPoolID, "pool-123"},
		{"Cognito.AppClientID", cfg.Cognito.AppClientID, "client-456"},
		{"Cognito.AppClientSecret", cfg.Cognito.AppClientSecret, "secret-789"},
		{"LogLevel", cfg.LogLevel, "debug"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %s, want %s", tt.got, tt.want)
			}
		})
	}
}

func TestAuthDevMode_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"lowercase true", "true", true},
		{"uppercase TRUE", "TRUE", true},
		{"mixed case True", "True", true},
		{"lowercase false", "false", false},
		{"uppercase FALSE", "FALSE", false},
		{"empty", "", false},
		{"random string", "yes", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("AUTH_DEV_MODE", tt.value)

			cfg := config.Load()
			if cfg.AuthDevMode != tt.want {
				t.Errorf("AUTH_DEV_MODE=%q: got %v, want %v", tt.value, cfg.AuthDevMode, tt.want)
			}
		})
	}
}

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantSub  string
	}{
		{
			name:     "simple password",
			password: "todo",
			wantSub:  "todo:todo@",
		},
		{
			name:     "password with special chars",
			password: "p@ss/w#rd?",
			wantSub:  "todo:p%40ss%2Fw%23rd%3F@",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DB_PASSWORD", tt.password)

			cfg := config.Load()
			dsn := cfg.DB.DSN()

			if !strings.Contains(dsn, tt.wantSub) {
				t.Errorf("DSN=%s, want to contain %s", dsn, tt.wantSub)
			}
			if !strings.HasPrefix(dsn, "postgres://") {
				t.Errorf("DSN=%s, want postgres:// prefix", dsn)
			}
			if !strings.Contains(dsn, "sslmode=disable") {
				t.Errorf("DSN=%s, want sslmode=disable", dsn)
			}
		})
	}
}

func TestConfig_ParseLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  slog.Level
	}{
		{"debug", "debug", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"uppercase DEBUG", "DEBUG", slog.LevelDebug},
		{"mixed case Warn", "Warn", slog.LevelWarn},
		{"empty defaults to info", "", slog.LevelInfo},
		{"invalid defaults to info", "verbose", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("LOG_LEVEL", tt.value)

			cfg := config.Load()
			got := cfg.ParseLogLevel()

			if got != tt.want {
				t.Errorf("LOG_LEVEL=%q: got %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name       string
		port       string
		env        string
		devMode    string
		poolID     string
		clientID   string
		wantErr    string
	}{
		{"valid local dev mode", "8080", "local", "true", "", "", ""},
		{"valid alpha", "8080", "alpha", "false", "pool-1", "client-1", ""},
		{"valid beta", "9090", "beta", "false", "pool-1", "client-1", ""},
		{"valid prod", "80", "prod", "false", "pool-1", "client-1", ""},
		{"invalid port", "abc", "local", "false", "", "", "invalid SERVER_PORT"},
		{"invalid env", "8080", "staging", "false", "", "", "invalid APP_ENV"},
		{"dev mode in alpha", "8080", "alpha", "true", "", "", "AUTH_DEV_MODE must not be enabled"},
		{"dev mode in beta", "8080", "beta", "true", "", "", "AUTH_DEV_MODE must not be enabled"},
		{"dev mode in prod", "8080", "prod", "true", "", "", "AUTH_DEV_MODE must not be enabled"},
		{"missing pool id non-dev", "8080", "local", "false", "", "client-1", "COGNITO_USER_POOL_ID is required"},
		{"missing client id non-dev", "8080", "local", "false", "pool-1", "", "COGNITO_APP_CLIENT_ID is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("SERVER_PORT", tt.port)
			t.Setenv("APP_ENV", tt.env)
			t.Setenv("AUTH_DEV_MODE", tt.devMode)
			if tt.poolID != "" {
				t.Setenv("COGNITO_USER_POOL_ID", tt.poolID)
			}
			if tt.clientID != "" {
				t.Setenv("COGNITO_APP_CLIENT_ID", tt.clientID)
			}

			cfg := config.Load()
			err := cfg.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
