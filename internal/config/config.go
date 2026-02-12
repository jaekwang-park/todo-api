package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var validEnvs = map[string]bool{
	"local": true,
	"alpha": true,
	"beta":  true,
	"prod":  true,
}

type Config struct {
	ServerPort  string
	AppEnv      string
	AuthDevMode bool
	DB          DBConfig
	Cognito     CognitoConfig
}

func (c Config) Validate() error {
	if _, err := strconv.Atoi(c.ServerPort); err != nil {
		return fmt.Errorf("invalid SERVER_PORT %q: %w", c.ServerPort, err)
	}
	if !validEnvs[c.AppEnv] {
		return fmt.Errorf("invalid APP_ENV %q: must be one of local, alpha, beta, prod", c.AppEnv)
	}
	if c.AuthDevMode && c.AppEnv != "local" {
		return fmt.Errorf("AUTH_DEV_MODE must not be enabled in %s environment", c.AppEnv)
	}
	return nil
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, d.Port),
		Path:     d.Name,
		RawQuery: fmt.Sprintf("sslmode=%s", url.QueryEscape(d.SSLMode)),
	}
	return u.String()
}

type CognitoConfig struct {
	Region      string
	UserPoolID  string
	AppClientID string
}

func Load() Config {
	return Config{
		ServerPort:  envOrDefault("SERVER_PORT", "8080"),
		AppEnv:      envOrDefault("APP_ENV", "local"),
		AuthDevMode: strings.EqualFold(envOrDefault("AUTH_DEV_MODE", "false"), "true"),
		DB: DBConfig{
			Host:     envOrDefault("DB_HOST", "localhost"),
			Port:     envOrDefault("DB_PORT", "5432"),
			User:     envOrDefault("DB_USER", "todo"),
			Password: envOrDefault("DB_PASSWORD", "todo"),
			Name:     envOrDefault("DB_NAME", "todo"),
			SSLMode:  envOrDefault("DB_SSLMODE", "disable"),
		},
		Cognito: CognitoConfig{
			Region:      envOrDefault("COGNITO_REGION", "ap-northeast-2"),
			UserPoolID:  os.Getenv("COGNITO_USER_POOL_ID"),
			AppClientID: os.Getenv("COGNITO_APP_CLIENT_ID"),
		},
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
