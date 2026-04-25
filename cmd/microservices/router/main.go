package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"

	"github.com/shanth1/morphic-monad/internal/modules/router"
	"github.com/shanth1/morphic-monad/internal/modules/router/adapters/classifier"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

const (
	AppName       = "morphic-monad"
	ServiceRouter = "router-svc"
)

func main() {
	baseLogger := log.New()

	cfg, err := config.Load()
	if err != nil {
		baseLogger.Fatal().Err(err).Msg("failed to load configuration")
	}

	if err := cfg.Validate(); err != nil {
		baseLogger.Fatal().Err(err).Msg("configuration validation failed")
	}

	logger := baseLogger.WithOptions(log.WithConfig(log.Config{
		Level:        cfg.Logger.Level,
		App:          AppName,
		Service:      ServiceRouter,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.System.Env != consts.EnvProd,
		JSONOutput:   cfg.System.Env == consts.EnvProd,
	}))

	logger.Info().
		Any(logkeys.Env, cfg.System.Env).
		Str(logkeys.GitHash, CommitHash).
		Msg("initializing router microservice")

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. INFRASTRUCTURE
	busClient, err := bus.NewClient(
		ServiceRouter,
		cfg.Transport.Nats.URL,
		cfg.Transport.Nats.StreamName,
		logger.With(log.Str("component", "nats_client")),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to external NATS")
	}
	defer busClient.Close()

	if err := busClient.InitStream(appCtx); err != nil {
		logger.Fatal().Err(err).Msg("failed to init JetStream stream")
	}

	// 2. ROUTER
	// Selecting a strategy based on the configuration
	var ruleEngine router.Classifier
	if cfg.Modules.Router.Strategy == "static" {
		ruleEngine = classifier.NewStaticRuleEngine() // TODO: cfg.Modules.Router.StaticRules
	} else {
		// LLM Router
		ruleEngine = classifier.NewStaticRuleEngine()
	}

	routerCore := router.NewService(busClient, busClient, ruleEngine, logger.With(log.Str("module", "router")))

	// 3. ORCHESTRATION
	supervisor := app.NewSupervisor(logger)
	supervisor.Register(routerCore)

	logger.Info().Msg("router service started successfully")

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("router service terminated with error")
	}
}
