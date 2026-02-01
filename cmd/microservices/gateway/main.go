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
		Service:      appconsts.ServiceGateway,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.System.Env != consts.EnvProd,
		JSONOutput:   cfg.System.Env == consts.EnvProd,
	}))
	logger = logger.With(log.Str(logkeys.Component, appconsts.ServiceGateway))

	logger.Info().
		Any(logkeys.Env, cfg.System.Env).
		Str(logkeys.GitHash, CommitHash).
		Str(logkeys.BuildTime, BuildTime).
		Str(logkeys.Service, appconsts.ServiceGateway).
		Msg(logmsg.AppInitializing)

	supervisor := app.New(cfg, logger)
	bus, err := natsclient.New(appconsts.ServiceGateway, cfg.Transport.Nats.URL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusFailed)
	}
	gateway := gateway.New(bus, logger)
	supervisor.Register(gateway)

	appCtx, cancel := ctx.GetAppCtx()
	defer cancel()

	logger.Info().Msg(logmsg.AppStarting)
	if err := supervisor.Run(appCtx); err != nil && err != context.Canceled {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
