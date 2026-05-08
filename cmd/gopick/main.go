package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/awfufu/gopick/internal/app"
	"github.com/awfufu/gopick/internal/config"
)

func main() {
	configPath := flag.String("f", "config.yml", "path to config file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	a := app.New(cfg)

	if err := a.Run(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
