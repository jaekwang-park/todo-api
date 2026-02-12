package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/jaekwang-park/todo-api/internal/config"
	todohttp "github.com/jaekwang-park/todo-api/internal/http"
	"github.com/jaekwang-park/todo-api/internal/repository"
	"github.com/jaekwang-park/todo-api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(context.Background(), logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	logger.Info("config loaded",
		"env", cfg.AppEnv,
		"port", cfg.ServerPort,
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

	// Services
	todoSvc := service.NewTodoService(todoRepo)

	// HTTP Server
	srv := todohttp.NewServer(cfg.ServerPort, logger, todoSvc)

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
