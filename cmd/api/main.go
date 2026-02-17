package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	cognitopkg "github.com/jaekwang-park/todo-api/internal/cognito"
	"github.com/jaekwang-park/todo-api/internal/config"
	todohttp "github.com/jaekwang-park/todo-api/internal/http"
	"github.com/jaekwang-park/todo-api/internal/middleware"
	"github.com/jaekwang-park/todo-api/internal/model"
	"github.com/jaekwang-park/todo-api/internal/repository"
	"github.com/jaekwang-park/todo-api/internal/service"
)

// userResolverAdapter adapts a user repository to the middleware.UserResolver interface.
type userResolverAdapter struct {
	repo interface {
		GetByCognitoSub(ctx context.Context, cognitoSub string) (model.User, error)
	}
}

func (a *userResolverAdapter) ResolveUserID(ctx context.Context, cognitoSub string) (string, error) {
	user, err := a.repo.GetByCognitoSub(ctx, cognitoSub)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", middleware.ErrUserNotFound
		}
		return "", fmt.Errorf("failed to resolve user: %w", err)
	}
	return user.ID, nil
}

func main() {
	// Initial logger at info level; reconfigured after config load
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(context.Background()); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.ParseLogLevel(),
	}))
	slog.SetDefault(logger)

	logger.Info("config loaded",
		"env", cfg.AppEnv,
		"port", cfg.ServerPort,
		"auth_dev_mode", cfg.AuthDevMode,
		"log_level", cfg.LogLevel,
	)

	// Database connection
	db, err := repository.NewDB(cfg.DB.DSN())
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("database connected")

	// Repositories
	todoRepo := repository.NewPostgresTodo(db)
	userRepo := repository.NewPostgresUser(db)

	// Services
	todoSvc := service.NewTodoService(todoRepo)

	// Cognito client + Auth service
	var authSvc *service.AuthService
	if cfg.Cognito.AppClientID != "" {
		cognitoClient, err := cognitopkg.NewAWSClient(
			ctx,
			cfg.Cognito.Region,
			cfg.Cognito.AppClientID,
			cfg.Cognito.AppClientSecret,
		)
		if err != nil {
			return err
		}
		authSvc = service.NewAuthService(cognitoClient, userRepo)
		logger.Info("cognito client initialized", "region", cfg.Cognito.Region)
	} else {
		logger.Warn("cognito client not initialized: COGNITO_APP_CLIENT_ID not set")
	}

	// Auth middleware
	authCfg := middleware.AuthConfig{
		DevMode: cfg.AuthDevMode,
	}
	if !cfg.AuthDevMode {
		jwksURL := middleware.CognitoJWKSURL(cfg.Cognito.Region, cfg.Cognito.UserPoolID)
		authCfg.JWKSClient = middleware.NewJWKSClient(jwksURL)
		authCfg.Issuer = middleware.CognitoIssuer(cfg.Cognito.Region, cfg.Cognito.UserPoolID)
		authCfg.AppClientID = cfg.Cognito.AppClientID
		authCfg.UserResolver = &userResolverAdapter{repo: userRepo}
	}
	auth, err := middleware.NewAuth(authCfg)
	if err != nil {
		return fmt.Errorf("failed to create auth middleware: %w", err)
	}

	// HTTP Server
	srv := todohttp.NewServer(cfg.ServerPort, logger, todoSvc, authSvc, auth)

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	logger.Info("server starting", "port", cfg.ServerPort)

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server stopped gracefully")
	return nil
}
