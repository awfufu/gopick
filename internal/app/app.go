package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/httpserver"
	"github.com/awfufu/gopick/internal/maiyatian"
	"github.com/awfufu/gopick/internal/service"
)

type App struct {
	server *httpserver.Server
	logger *slog.Logger
}

func New(cfg config.Config) *App {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	maiyatianClient := maiyatian.NewHTTPClient(cfg.Maiyatian)
	orderService := service.NewOrderService(maiyatianClient)
	server := httpserver.New(cfg.HTTP, logger, orderService)

	return &App{
		server: server,
		logger: logger,
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.server.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		a.logger.Info("shutting down")
		return a.server.Shutdown(shutdownCtx)
	}
}
