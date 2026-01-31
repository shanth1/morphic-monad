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
	"github.com/shanth1/morphic-monad/internal/modules/router"
	appconsts "github.com/shanth1/morphic-monad/internal/pkg/consts"
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

	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg(logmsg.ValidatingConfigFailed)
	}

	logger = logger.WithOptions(log.WithConfig(log.Config{
		Level:        cfg.Logger.Level,
		App:          appconsts.AppName,
		Service:      appconsts.ServiceRouter,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.App.Env != consts.EnvProd,
		JSONOutput:   cfg.App.Env == consts.EnvProd,
	}))
	logger = logger.With(log.Str(logkeys.Component, appconsts.ServiceRouter))

	logger.Info().
		Any(logkeys.Env, cfg.App.Env).
		Str(logkeys.GitHash, CommitHash).
		Str(logkeys.BuildTime, BuildTime).
		Str(logkeys.Service, appconsts.ServiceRouter).
		Msg(logmsg.AppInitializing)

	supervisor := app.New(cfg, logger)
	bus, err := natsclient.New(appconsts.ServiceRouter, cfg.Nats.URL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusFailed)
	}
	router := router.New(bus, logger)
	supervisor.Register(router)

	appCtx, cancel := ctx.GetAppCtx()
	defer cancel()

	logger.Info().Msg(logmsg.AppStarting)
	if err := supervisor.Run(appCtx); err != nil && err != context.Canceled {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
