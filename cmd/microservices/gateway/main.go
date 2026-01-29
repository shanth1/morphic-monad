package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	cfg.App.Mode = "microservices"

	logger := log.New().WithOptions(log.WithConfig(log.Config{
		Level: cfg.Logger.Level, JSONOutput: cfg.App.Env == consts.EnvProd,
	}))

	logger.Info().Msg("microservice [gateway] running...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
