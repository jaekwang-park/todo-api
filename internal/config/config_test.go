package config_test

import (
	"strings"
	"testing"

	"github.com/jaekwang-park/todo-api/internal/config"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SERVER_PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "DB_SSLMODE", "APP_ENV", "AUTH_DEV_MODE",
		"COGNITO_REGION", "COGNITO_USER_POOL_ID", "COGNITO_APP_CLIENT_ID",
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
		{"Cognito.Region", cfg.Cognito.Region, "ap-northeast-2"},
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

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		env     string
		devMode string
		wantErr string
	}{
		{"valid local", "8080", "local", "true", ""},
		{"valid alpha", "8080", "alpha", "false", ""},
		{"valid beta", "9090", "beta", "false", ""},
		{"valid prod", "80", "prod", "false", ""},
		{"invalid port", "abc", "local", "false", "invalid SERVER_PORT"},
		{"invalid env", "8080", "staging", "false", "invalid APP_ENV"},
		{"dev mode in alpha", "8080", "alpha", "true", "AUTH_DEV_MODE must not be enabled"},
		{"dev mode in beta", "8080", "beta", "true", "AUTH_DEV_MODE must not be enabled"},
		{"dev mode in prod", "8080", "prod", "true", "AUTH_DEV_MODE must not be enabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("SERVER_PORT", tt.port)
			t.Setenv("APP_ENV", tt.env)
			t.Setenv("AUTH_DEV_MODE", tt.devMode)

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
