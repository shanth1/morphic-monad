package main

import (
	"context"
	"errors"
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
	"github.com/shanth1/morphic-monad/internal/pkg/consts"
	"github.com/shanth1/morphic-monad/internal/pkg/logmsg"

	"github.com/shanth1/morphic-monad/internal/modules/workers/embedder"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

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
		Service:      consts.ServiceEmbedder,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.System.Env != goconsts.EnvProd,
		JSONOutput:   cfg.System.Env == goconsts.EnvProd,
	}))

	logger.Info().
		Any(logkeys.Env, cfg.System.Env).
		Str(logkeys.GitHash, CommitHash).
		Msg(logmsg.AppInitializing)

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// --- Infrastructure ---

	busClient, err := bus.NewClient(
		consts.ServiceEmbedder,
		cfg.Transport.Nats.URL,
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

	embedderWorker := embedder.NewService(
		busClient,
		busClient,
		memoryBlobStore,
		logger.With(log.Str("module", consts.ServiceEmbedder)),
	)

	supervisor := app.NewSupervisor(logger)
	supervisor.Register(embedderWorker)

	logger.Info().Msg(logmsg.AppStarting)

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
