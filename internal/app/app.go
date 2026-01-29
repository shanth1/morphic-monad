package app

import (
	"context"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/internal/config"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"golang.org/x/sync/errgroup"
)

type Supervisor struct {
	logger  log.Logger
	workers []ports.Worker
	cleanup []func()
}

func New(cfg *config.Config, logger log.Logger) *Supervisor {
	return &Supervisor{
		logger:  logger,
		workers: make([]ports.Worker, 0),
	}
}

func (s *Supervisor) Register(w ...ports.Worker) {
	s.workers = append(s.workers, w...)
}

func (s *Supervisor) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, w := range s.workers {
		worker := w
		g.Go(func() error {
			return worker.Run(ctx)
		})
	}

	s.logger.Info().Int(logkeys.Amount, len(s.workers)).Msg("supervisor started all workers")

	err := g.Wait()

	if err != nil && err != context.Canceled {
		s.logger.Error().Err(err).Msg("supervisor stopped with error")
	} else {
		s.logger.Info().Msg("supervisor stopped gracefully")
	}

	return err
}
