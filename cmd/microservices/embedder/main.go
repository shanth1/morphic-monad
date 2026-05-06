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
	"github.com/shanth1/gotools/logkeys"
	infrahttp "github.com/shanth1/morphic-monad/internal/infra/http"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"
	"github.com/shanth1/morphic-monad/internal/pkg/consts"
	"github.com/shanth1/morphic-monad/internal/pkg/logmsg"

	"github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/s3"
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

	s3Config := s3.Config{
		Endpoint:        cfg.Modules.Tools.BlobStore.S3.Endpoint,
		Region:          cfg.Modules.Tools.BlobStore.S3.Region,
		AccessKeyID:     cfg.Modules.Tools.BlobStore.S3.AccessKey,
		SecretAccessKey: cfg.Modules.Tools.BlobStore.S3.SecretKey,
		BucketName:      cfg.Modules.Tools.BlobStore.S3.Bucket,
		UsePathStyle:    cfg.Modules.Tools.BlobStore.S3.UsePathStyle,
	}
	s3Adapter, err := s3.NewAdapter(context.Background(), s3Config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize S3 adapter")
	}

	embedderWorker := embedder.NewService(
		busClient,
		busClient,
		s3Adapter,
		logger.With(log.Str("module", consts.ServiceEmbedder)),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	httpServer := infrahttp.NewServer("8083", mux, logger)
	supervisor := app.NewSupervisor(logger)
	supervisor.Register(embedderWorker, httpServer)

	logger.Info().Msg(logmsg.AppStarting)

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg(logmsg.AppRuntimeError)
	}
}
