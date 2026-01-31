package gateway

import (
	"context"
	"time"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type Service struct {
	bus    ports.Bus
	logger log.Logger
}

func New(bus ports.Bus, l log.Logger) *Service {
	return &Service{
		bus:    bus,
		logger: l,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	s.logger.Info().Msg("staring gateway service")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			payload := map[string]string{
				"content": "Hello World from Gateway",
				"source":  "simulation",
			}

			event, err := envelope.New("tenant-default", "data.text.ingest", payload)
			if err != nil {
				s.logger.Error().Err(err).Msg("creating envelope")
				continue
			}

			s.logger.Info().Str("event_id", event.ID).Msg("publishing event")
			if err := s.bus.Publish(ctx, "data.raw", event); err != nil {
				s.logger.Error().Str("event_id", event.ID).Err(err).Msg("publishing event")
			}
		}
	}
}
