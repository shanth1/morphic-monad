package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"
	infrahttp "github.com/shanth1/morphic-monad/internal/infra/http"

	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	gatewayhttp "github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/http"
	"github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/s3"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

const (
	AppName        = "morphic-monad"
	ServiceGateway = "gateway-svc"
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
		Service:      ServiceGateway,
		UDPAddress:   cfg.Logger.UDPAddress,
		EnableCaller: cfg.Logger.EnableCaller,
		Console:      cfg.System.Env != consts.EnvProd,
		JSONOutput:   cfg.System.Env == consts.EnvProd,
	}))

	logger.Info().
		Any(logkeys.Env, cfg.System.Env).
		Str(logkeys.GitHash, CommitHash).
		Msg("initializing gateway microservice")

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. INFRASTRUCTURE

	// Connecting to an external NATS cluster
	busClient, err := bus.NewClient(
		ServiceGateway,
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

	// Connecting to external AWS S3 / LocalStack
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

	// 2. GATEWAY
	gatewayCore := gateway.NewService(busClient, s3Adapter, logger.With(log.Str("module", "gateway")))
	gatewayHandler := gatewayhttp.NewHandler(gatewayCore, logger.With(log.Str("module", "gateway_http")))

	// 3. TRANSPORT
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gatewayHandler.HandleIngest)
	httpServer := infrahttp.NewServer(cfg.Modules.Gateway.Port, mux, logger.With(log.Str("component", "http_server")))

	// 4. ORCHESTRATION
	supervisor := app.NewSupervisor(logger)
	supervisor.Register(httpServer)

	logger.Info().Msg("gateway service started successfully")

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("gateway service terminated with error")
	}
}
