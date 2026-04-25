package router

import (
	"context"
	"encoding/json"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/internal/pkg/logmsg"
	"github.com/shanth1/morphic-monad/pkg/events"
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
	err := s.bus.Subscribe("data.raw", s.handleRawData, "router_group")
	if err != nil {
		return err
	}

	s.logger.Info().Str(logkeys.Topic, "data.raw").Msg("subscribed")
	<-ctx.Done()
	return nil
}

func (s *Service) handleRawData(ctx context.Context, event *events.Envelope) error {
	s.logger.Info().Str("event_id", event.ID).Str("tenant_id", event.TenantID).Str("type", event.Type).Msg("received event")

	var data map[string]string
	if err := json.Unmarshal(event.Payload, &data); err != nil {
		s.logger.Error().Err(err).Msg(logmsg.UnmarshallingFailed)
		return err
	}

	s.logger.Info().Int(logkeys.ContentLen, len(data["content"])).Msg("content")

	return nil
}
