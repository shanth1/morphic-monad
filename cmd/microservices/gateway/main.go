package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shanth1/gotools/log"
	infrahttp "github.com/shanth1/morphic-monad/internal/infra/http"

	"github.com/shanth1/morphic-monad/internal/app"
	"github.com/shanth1/morphic-monad/internal/infra/bus"
	"github.com/shanth1/morphic-monad/internal/infra/config"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	gatewayhttp "github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/http"
	"github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/s3"
)

func main() {
	logger := log.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// --- Инфраструктура ---
	busClient, err := bus.NewClient("gateway", cfg.Transport.Nats.URL, cfg.Transport.Nats.StreamName, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer busClient.Close()
	_ = busClient.InitStream(appCtx)

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

	// --- Инициализация Core ---
	gatewayCore := gateway.NewService(busClient, busClient, s3Adapter, logger)
	gatewayHandler := gatewayhttp.NewHandler(gatewayCore, logger)

	// --- Регистрация Роутов ---
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ingest", gatewayhttp.MetricsMiddleware("/v1/ingest", gatewayHandler.HandleIngest))
	mux.HandleFunc("/v1/search", gatewayhttp.MetricsMiddleware("/v1/search", gatewayHandler.HandleSearch))

	// НОВЫЕ РОУТЫ!
	mux.HandleFunc("/v1/events/stream", gatewayHandler.HandleStreamEvents)
	mux.HandleFunc("/v1/blob", gatewayHandler.HandleBlob)

	mux.Handle("/metrics", promhttp.Handler())

	httpServer := infrahttp.NewServer(cfg.Modules.Gateway.Port, mux, logger)

	supervisor := app.NewSupervisor(logger)
	supervisor.Register(httpServer, gatewayCore)

	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("runtime error")
	}
}
