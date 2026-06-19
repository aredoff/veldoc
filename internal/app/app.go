package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aredoff/veldoc/internal/auth"
	"github.com/aredoff/veldoc/internal/config"
	"github.com/aredoff/veldoc/internal/files"
	"github.com/aredoff/veldoc/internal/server"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	fileService, err := files.NewService(cfg.Root, cfg.MaxPreviewSize)
	if err != nil {
		return err
	}

	authenticator, err := auth.New(cfg)
	if err != nil {
		return err
	}

	srv := server.New(cfg, fileService, authenticator, logger)
	httpServer := srv.HTTPServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
