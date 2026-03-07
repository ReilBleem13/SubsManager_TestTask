package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/ReilBleem13/docs"
	"github.com/ReilBleem13/internal/config"
	"github.com/ReilBleem13/internal/infra/database"
	"github.com/ReilBleem13/internal/logger"
	"github.com/ReilBleem13/internal/repository"
	"github.com/ReilBleem13/internal/service"
	"github.com/ReilBleem13/internal/transport"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

// @title          Subscription Service API
// @version        1.0
// @description    API для управления подписками
// @contact.email  reilbleem@rambler.ru
// @host      localhost:8080
// @BasePath  /
// @schemes   http https
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger := logger.New(cfg.App.LogLevel)
	slog.SetDefault(logger)

	databaseClient, err := database.NewPostgres(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatal(err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	migrationsPath := filepath.Join("internal", "infra", "database", "migrations")
	if err := goose.Up(databaseClient.DB.DB, migrationsPath); err != nil {
		log.Fatal(err)
	}

	subRepo := repository.NewSubsRepo(ctx, databaseClient.DB)
	subSrv := service.NewSub(subRepo, logger)

	subHandler := transport.NewSubHandler(subSrv, logger)

	httpMux := transport.NewRouter(subHandler)
	httpAddr := ":" + cfg.App.Port
	httpServer := transport.NewServer(httpAddr, httpMux)

	httpErrCh := make(chan error, 1)

	go func() {
		logger.InfoContext(ctx, "Starting http server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			httpErrCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "Received shut down signal, starning graceful shutdown")
	case err := <-httpErrCh:
		logger.ErrorContext(ctx, "HTTP Server failed", "error", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "Error shutting down server", "error", err)
	}

	if err := databaseClient.DB.Close(); err != nil {
		logger.ErrorContext(shutdownCtx, "Error closing db", "error", err)
	}

	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.InfoContext(shutdownCtx, "Graceful shutdown timed out")
	} else {
		logger.InfoContext(shutdownCtx, "Graceful shutdown completed")
	}
}
