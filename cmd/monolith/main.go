package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	goconsts "github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"

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

	"github.com/shanth1/morphic-monad/internal/modules/workers/chunker"
	"github.com/shanth1/morphic-monad/internal/modules/workers/embedder"
	embedderllm "github.com/shanth1/morphic-monad/internal/modules/workers/embedder/adapters/llm"
	"github.com/shanth1/morphic-monad/internal/modules/workers/vision"
	visionllm "github.com/shanth1/morphic-monad/internal/modules/workers/vision/adapters/llm"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

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

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// --- Infrastructure ---
	embeddedNats, err := bus.NewServer(logger.With(log.Str("component", "nats_server")))
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize embedded NATS server")
	}
	if err := embeddedNats.Start(); err != nil {
		logger.Fatal().Err(err).Msg(logmsg.InitBusFailed)
	}

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

	memoryBlobStore := blob.NewMemoryStorage()
	memoryVectorDB := vectordb.NewMemoryVectorDB()

	// --- AI Adapters ---
	var embedAdapter embedder.TextVectoriser
	if cfg.Modules.Tools.Embedder.Provider == "ollama" {
		embedAdapter = embedderllm.NewOllamaVectoriser(cfg.Modules.Tools.Embedder.Ollama.BaseURL, cfg.Modules.Tools.Embedder.Ollama.Model)
	} else {
		embedAdapter = embedderllm.NewMockVectoriser(384)
	}

	var visionAdapter vision.ImageDescriber
	if cfg.Modules.Tools.Vision.Provider == "ollama" {
		visionAdapter = visionllm.NewOllamaDescriber(cfg.Modules.Tools.Vision.Ollama.BaseURL, cfg.Modules.Tools.Vision.Ollama.Model)
	} else {
		visionAdapter = visionllm.NewMockDescriber()
	}

	// --- Core Modules ---
	engineCore := engine.NewService(busClient, busClient, memoryVectorDB, logger.With(log.Str("module", consts.ServiceEngine)))
	gatewayCore := gateway.NewService(busClient, busClient, memoryBlobStore, logger.With(log.Str("module", consts.ServiceGateway)))
	routerCore := router.NewService(busClient, busClient, classifier.NewStaticRuleEngine(), logger.With(log.Str("module", consts.ServiceRouter)))

	// --- Worker Modules ---
	visionWorker := vision.NewService(busClient, busClient, memoryBlobStore, visionAdapter, logger.With(log.Str("module", "vision")))
	chunkerWorker := chunker.NewService(busClient, busClient, memoryBlobStore, logger.With(log.Str("module", "chunker")))
	embedWorker := embedder.NewService(busClient, busClient, memoryBlobStore, embedAdapter, logger.With(log.Str("module", consts.ServiceEmbedder)))

	// --- Transport ---
	gatewayHandler := gatewayhttp.NewHandler(gatewayCore, logger.With(log.Str("module", consts.ServiceGateway)))

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gatewayhttp.MetricsMiddleware("/v1/ingest", gatewayHandler.HandleIngest))
	mux.HandleFunc("/v1/search", gatewayhttp.MetricsMiddleware("/v1/search", gatewayHandler.HandleSearch))
	mux.HandleFunc("/v1/events/stream", gatewayHandler.HandleStreamEvents)
	mux.HandleFunc("/v1/blob", gatewayHandler.HandleBlob)
	mux.Handle("/metrics", promhttp.Handler())

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
		visionWorker,
		chunkerWorker,
		embedWorker,
	)

	logger.Info().Msg(logmsg.AppStarting)

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
