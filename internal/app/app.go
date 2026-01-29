package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsclient"
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsembed"
	"github.com/shanth1/morphic-monad/internal/modules/gateway"
	"github.com/shanth1/morphic-monad/internal/modules/router"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg    *config.Config
	logger log.Logger
	bus    ports.Bus
	server *natsembed.Server
}

func New(cfg *config.Config, logger log.Logger) (*App, func(), error) {
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	var natsServer *natsembed.Server
	var err error
	switch cfg.App.Mode {
	case "monolith":
		natsServer, err = natsembed.New()
		if err != nil {
			return nil, cleanup, fmt.Errorf("embed nats error: %w", err)
		}
		cleanups = append(cleanups, func() {
			natsServer.Shutdown()
		})
	default:
		return nil, cleanup, fmt.Errorf("invalid app mode: %s", cfg.App.Mode)
	}

	bus, err := natsclient.New(cfg.Nats.URL)
	if err != nil {
		return nil, cleanup, fmt.Errorf("bus connection error: %w", err)
	}
	cleanups = append(cleanups, func() {
		bus.Close()
	})

	gatewaySvc := gateway.New(bus)
	routerSvc := router.New(bus)

	routerSvc.Start()
	gatewaySvc.EmulateIngest()

	return &App{
		cfg:    cfg,
		logger: logger,
		bus:    bus,
		server: natsServer,
	}, cleanup, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		a.server.Start()

		return nil
	})

	<-gCtx.Done()

	a.logger.Info().Msg("shutting down application...")

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	a.logger.Info().Msg("application stopped gracefully")
	return nil
}
