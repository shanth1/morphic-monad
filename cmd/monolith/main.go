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
	"github.com/shanth1/morphic-monad/internal/infra/vectordb"
	"github.com/shanth1/morphic-monad/internal/pkg/consts"
	"github.com/shanth1/morphic-monad/internal/pkg/logmsg"

	"github.com/shanth1/morphic-monad/internal/modules/engine"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	gatewayhttp "github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/http"
	"github.com/shanth1/morphic-monad/internal/modules/router"
	"github.com/shanth1/morphic-monad/internal/modules/router/adapters/classifier"
	"github.com/shanth1/morphic-monad/internal/modules/workers/embedder"
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
	baseLogger := log.New()

	cfg, err := config.Load()
	if err != nil {
		baseLogger.Fatal().Err(err).Msg(logmsg.LoadConfigFailed)
	}

	if err := cfg.Validate(); err != nil {
		baseLogger.Fatal().Err(err).Msg(logmsg.ValidatingConfigFailed)
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
		Str(logkeys.BuildTime, BuildTime).
		Msg(logmsg.AppInitializing)

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// --- Infrastructure ---

	// Built-in NATS Server
	embeddedNats, err := bus.NewServer(logger.With(log.Str("component", "nats_server")))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize embedded NATS server")
	}
	if err := embeddedNats.Start(); err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusFailed)
	}

	// Single NATS Client for all modules
	busClient, err := bus.NewClient(
		consts.ServiceMonolith,
		embeddedNats.URL(),
		cfg.Transport.Nats.StreamName,
		logger.With(log.Str("component", consts.ComponentNATSClient)),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg(logmsg.BusConnectionFailed)
	}
	defer busClient.Close()

	if err := busClient.InitStream(appCtx); err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusStreamFailed)
	}

	// In-Memory Blob Store
	memoryBlobStore := blob.NewMemoryStorage()

	// In-Memory VectorDB
	memoryVectorDB := vectordb.NewMemoryVectorDB()

	// --- Core Modules ---

	engineCore := engine.NewService(
		busClient,
		busClient,
		memoryVectorDB,
		logger.With(log.Str("module", consts.ServiceEngine)),
	)

	gatewayCore := gateway.NewService(
		busClient,
		busClient,
		memoryBlobStore,
		logger.With(log.Str("module", consts.ServiceGateway)),
	)

	staticClassifier := classifier.NewStaticRuleEngine()
	routerCore := router.NewService(
		busClient,
		busClient,
		staticClassifier,
		logger.With(log.Str("module", consts.ServiceRouter)),
	)

	// --- Worker Modules ---

	embeddedWorker := embedder.NewService(
		busClient,       // EventSubscriber
		busClient,       // EventPublisher
		memoryBlobStore, // BlobReader
		logger.With(log.Str("module", consts.ServiceEmbedder)),
	)

	// --- Transport ---

	gatewayHandler := gatewayhttp.NewHandler(
		gatewayCore,
		logger.With(log.Str("module", consts.ServiceGateway)),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gatewayHandler.HandleIngest)
	mux.HandleFunc("/v1/search", gatewayHandler.HandleSearch)
	httpServer := infrahttp.NewServer(
		cfg.Modules.Gateway.Port,
		mux,
		logger.With(log.Str("component", consts.ComponentHTTPServer)),
	)

	supervisor := app.NewSupervisor(logger)
	supervisor.Register(
		&natsRunner{srv: embeddedNats},
		httpServer,
		routerCore,
		gatewayCore,
		engineCore,
		embeddedWorker,
	)

	logger.Info().Msg(logmsg.AppStarting)

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
