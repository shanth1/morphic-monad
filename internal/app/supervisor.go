package app

import (
	"context"

	"github.com/shanth1/gotools/log"
	"golang.org/x/sync/errgroup"
)

// Runnable is an interface for any module that needs to run in the background.
type Runnable interface {
	Start(ctx context.Context) error
}

type Supervisor struct {
	services []Runnable
	logger   log.Logger
}

func NewSupervisor(logger log.Logger) *Supervisor {
	return &Supervisor{
		logger: logger,
	}
}

// Register adds modules to the run queue.
func (s *Supervisor) Register(services ...Runnable) {
	s.services = append(s.services, services...)
}

// Run starts all modules and blocks the main thread until it receives a stop signal.
func (s *Supervisor) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, svc := range s.services {
		svc := svc
		g.Go(func() error {
			return svc.Start(gCtx)
		})
	}

	s.logger.Info().Msg("platform started, waiting for termination signal...")

	if err := g.Wait(); err != nil {
		s.logger.Error().Err(err).Msg("platform stopped with error")
		return err
	}

	s.logger.Info().Msg("platform stopped gracefully")
	return nil
}
