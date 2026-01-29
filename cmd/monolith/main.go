package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	"github.com/shanth1/morphic-monad/internal/modules/router"
)

func main() {
	logger := log.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("load config failed")
	}

	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}

	logger = logger.WithOptions(log.WithConfig(log.Config{
		Level:        cfg.Logger.Level,
		App:          cfg.Logger.App,
		Service:      cfg.Logger.Service,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.App.Env != consts.EnvProd,
		JSONOutput:   cfg.App.Env == consts.EnvProd,
	}))

	logger.Info().
		Any(logkeys.Env, cfg.App.Env).
		Msg("application initializing...")

	container, err := app.Bootstrap(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("bootstrap failed")
	}
	defer container.Shutdown()

	gatewaySvc := gateway.New(container.Bus)
	routerSvc := router.New(container.Bus)

	routerSvc.Start()
	gatewaySvc.EmulateIngest()

	logger.Info().Msg("monolith running")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info().Msg("shutting down...")
}
