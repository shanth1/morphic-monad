package main

import (
	"context"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/ctx"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsclient"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	appconsts "github.com/shanth1/morphic-monad/internal/pkg/consts"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
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
		Str(logkeys.GitHash, CommitHash).
		Str(logkeys.BuildTime, BuildTime).
		Str(logkeys.Service, appconsts.ServiceGateway).
		Msg("application initializing...")

	supervisor := app.New(cfg, logger)
	bus, err := natsclient.New(cfg.Nats.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init bus client")
	}
	gateway := gateway.New(bus)
	supervisor.Register(gateway)

	appCtx, cancel := ctx.GetAppCtx()
	defer cancel()

	logger.Info().Msg("starting application...")
	if err := supervisor.Run(appCtx); err != nil && err != context.Canceled {
		logger.Fatal().Err(err).Msg("application runtime error")
	}
}
