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
	"github.com/shanth1/morphic-monad/internal/modules/gateway/adapters/s3"

	"github.com/shanth1/morphic-monad/internal/modules/workers/vision"
	visionllm "github.com/shanth1/morphic-monad/internal/modules/workers/vision/adapters/llm"
)

func main() {
	logger := log.New()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	busClient, err := bus.NewClient("vision", cfg.Transport.Nats.URL, cfg.Transport.Nats.StreamName, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	defer busClient.Close()
	_ = busClient.InitStream(appCtx)

	s3Config := s3.Config{
		Endpoint:        cfg.Modules.Tools.BlobStore.S3.Endpoint,
		BucketName:      cfg.Modules.Tools.BlobStore.S3.Bucket,
		AccessKeyID:     cfg.Modules.Tools.BlobStore.S3.AccessKey,
		SecretAccessKey: cfg.Modules.Tools.BlobStore.S3.SecretKey,
		Region:          cfg.Modules.Tools.BlobStore.S3.Region,
		UsePathStyle:    cfg.Modules.Tools.BlobStore.S3.UsePathStyle,
	}
	s3Adapter, err := s3.NewAdapter(context.Background(), s3Config)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to S3")
	}

	var visionAdapter vision.ImageDescriber
	if cfg.Modules.Tools.Vision.Provider == "ollama" {
		visionAdapter = visionllm.NewOllamaDescriber(cfg.Modules.Tools.Vision.Ollama.BaseURL, cfg.Modules.Tools.Vision.Ollama.Model)
	} else {
		visionAdapter = visionllm.NewMockDescriber()
	}

	visionWorker := vision.NewService(busClient, busClient, s3Adapter, visionAdapter, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	httpServer := infrahttp.NewServer("8084", mux, logger)

	supervisor := app.NewSupervisor(logger)
	supervisor.Register(visionWorker, httpServer)
	if err := supervisor.Run(appCtx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal().Err(err).Msg("runtime error")
	}
}
