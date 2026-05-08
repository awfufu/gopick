package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/httpserver"
	"github.com/awfufu/gopick/internal/logging"
	"github.com/awfufu/gopick/internal/maiyatian"
	"github.com/awfufu/gopick/internal/myshop"
	"github.com/awfufu/gopick/internal/service"
	"github.com/awfufu/gopick/internal/wsclient"
)

type App struct {
	server   *httpserver.Server
	logger   *slog.Logger
	wsClient *wsclient.MaiyatianWSClient
}

func New(cfg config.Config) *App {
	logger := slog.New(logging.NewHandler(os.Stdout, slog.LevelInfo))

	maiyatianClient := maiyatian.NewHTTPClient(cfg.Maiyatian)
	orderService := service.NewOrderService(maiyatianClient)
	myshopClient := myshop.NewClient(cfg.Upload)
	reporter := service.NewNewOrderReporter(maiyatianClient, myshopClient)
	ws := wsclient.NewMaiyatianWSClient(logger, maiyatianClient, cfg.Maiyatian, reporter)
	server := httpserver.New(cfg.HTTP, logger, orderService, ws)

	return &App{
		server:   server,
		logger:   logger,
		wsClient: ws,
	}
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.server.Start()
	}()

	go a.wsClient.Run(ctx)

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
