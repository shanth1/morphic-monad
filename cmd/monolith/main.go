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
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsembed"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	"github.com/shanth1/morphic-monad/internal/modules/router"
	"github.com/shanth1/morphic-monad/internal/pkg/logmsg"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

func main() {
	logger := log.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.LoadConfigFailed)
	}

	// TODO: default nats url in config
	if cfg.Nats.URL == "" {
		cfg.Nats.URL = "nats://127.0.0.1:4222"
	}

	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg(logmsg.ValidatingConfigFailed)
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
		Msg(logmsg.AppInitializing)

	supervisor := app.New(cfg, logger)

	busServer, err := natsembed.New()
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusServerFailed)
	}
	if err := busServer.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start embed nats")
	}

	supervisor.Register(busServer)

	bus, err := natsclient.New(cfg.Nats.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusFailed)
	}
	gateway := gateway.New(bus)
	router := router.New(bus)

	supervisor.Register(gateway, router)

	appCtx, cancel := ctx.GetAppCtx()
	defer cancel()

	logger.Info().Msg(logmsg.AppStarting)
	if err := supervisor.Run(appCtx); err != nil && err != context.Canceled {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
