package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	goconsts "github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/blob"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"
	infrahttp "github.com/shanth1/morphic-monad/internal/infra/http"
	"github.com/shanth1/morphic-monad/internal/pkg/consts"

	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	gatewayhttp "github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/http"
	"github.com/shanth1/morphic-monad/internal/modules/router"
	"github.com/shanth1/morphic-monad/internal/modules/router/adapters/classifier"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

// natsRunner adapts the built-in NATS server to the app.Runnable interface
type natsRunner struct {
	srv *bus.Server
}

func (n *natsRunner) Start(ctx context.Context) error {
	return n.srv.Run(ctx)
}

func main() {
	// 1. INITIALIZATION
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
		App:          consts.AppName,
		Service:      consts.ServiceMonolith,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.System.Env != goconsts.EnvProd,
		JSONOutput:   cfg.System.Env == goconsts.EnvProd,
	}))

	logger.Info().
		Any(logkeys.Env, cfg.System.Env).
		Str(logkeys.GitHash, CommitHash).
		Str("build_time", BuildTime).
		Msg("initializing monolith application")

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 2. INFRASTRUCTURE (IN-MEMORY ADAPTERS)

	// Built-in NATS Server
	embeddedNats, err := bus.NewServer(logger.With(log.Str("component", "nats_server")))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize embedded NATS server")
	}
	if err := embeddedNats.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start embedded NATS server")
	}

	// Single NATS Client for all modules
	busClient, err := bus.NewClient(
		consts.ServiceMonolith,
		embeddedNats.URL(),
		cfg.Transport.Nats.StreamName,
		logger.With(log.Str("component", "nats_client")),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer busClient.Close()

	if err := busClient.InitStream(appCtx); err != nil {
		logger.Fatal().Err(err).Msg("failed to init JetStream stream")
	}

	// In-Memory Blob Store (replaces S3)
	memoryBlobStore := blob.NewMemoryStorage()

	// 3. MODULES

	// Gateway
	gatewayCore := gateway.NewService(busClient, memoryBlobStore, logger.With(log.Str("module", "gateway")))
	gatewayHandler := gatewayhttp.NewHandler(gatewayCore, logger.With(log.Str("module", consts.ServiceGateway)))

	// Router
	staticClassifier := classifier.NewStaticRuleEngine()
	routerCore := router.NewService(busClient, busClient, staticClassifier, logger.With(log.Str("module", consts.ServiceRouter)))

	// 4. TRANSPORT (HTTP)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gatewayHandler.HandleIngest)
	httpServer := infrahttp.NewServer(cfg.Modules.Gateway.Port, mux, logger.With(log.Str("component", "http_server")))

	// 5. СУПЕРВИЗОР
	supervisor := app.NewSupervisor(logger)
	supervisor.Register(
		&natsRunner{srv: embeddedNats},
		httpServer,
		routerCore,
	)

	logger.Info().Msg("platform started successfully")

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("platform terminated with error")
	}
}
