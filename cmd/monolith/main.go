package main

import (
	"fmt"
	"os"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/ctx"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/config"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	logger := log.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Error().Err(err).Msg("load config failed")
		return err
	}

	fmt.Println("TEST:", cfg)
	if err := cfg.Validate(); err != nil {
		logger.Error().Err(err).Msg("invalid configuration")
		return err
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

	application, cleanup, err := app.New(cfg, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to init app")
		return err
	}
	defer cleanup()

	appCtx, cancel := ctx.GetAppCtx()
	defer cancel()

	logger.Info().Msg("starting application...")
	if err := application.Run(appCtx); err != nil {
		logger.Error().Err(err).Msg("application runtime error")
		return err
	}

	return nil
}
