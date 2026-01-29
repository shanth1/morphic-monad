package app

import (
	"fmt"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsclient"
	"github.com/shanth1/morphic-monad/internal/infra/transport/natsembed"
)

type Container struct {
	Cfg    *config.Config
	Logger log.Logger
	Bus    ports.Bus

	server  *natsembed.Server
	cleanup []func()
}

func Bootstrap(cfg *config.Config, logger log.Logger) (*Container, error) {
	c := &Container{
		Cfg:    cfg,
		Logger: logger,
	}

	if cfg.App.Mode == "monolith" {
		server, err := natsembed.New()
		if err != nil {
			return nil, fmt.Errorf("embed nats init error: %w", err)
		}
		server.Start()
		c.server = server
		c.cleanup = append(c.cleanup, func() { server.Shutdown() })
		logger.Info().Msg("embedded NATS started")
	}

	bus, err := natsclient.New(cfg.Nats.URL)
	if err != nil {
		c.Shutdown()
		return nil, fmt.Errorf("bus connection error: %w", err)
	}
	c.Bus = bus
	c.cleanup = append(c.cleanup, func() { bus.Close() })

	return c, nil
}

func (c *Container) Shutdown() {
	for i := len(c.cleanup) - 1; i >= 0; i-- {
		c.cleanup[i]()
	}
}
