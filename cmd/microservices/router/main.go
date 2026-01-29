package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/modules/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	cfg.App.Mode = "microservices"

	logger := log.New().WithOptions(log.WithConfig(log.Config{
		Level: cfg.Logger.Level, JSONOutput: cfg.App.Env == consts.EnvProd,
	}))

	container, err := app.Bootstrap(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("bootstrap failed")
	}
	defer container.Shutdown()

	svc := router.New(container.Bus)
	svc.Start()

	logger.Info().Msg("microservice [router] running...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
