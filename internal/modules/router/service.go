package router

import (
	"context"
	"log"

	"github.com/shanth1/morphic-monad/internal/core/ports"
	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type Service struct {
	bus ports.Bus
}

func New(bus ports.Bus) *Service {
	return &Service{bus: bus}
}

func (s *Service) Run(ctx context.Context) error {
	err := s.bus.Subscribe("data.raw", s.handleEvent, "router-workers")
	if err != nil {
		return err
	}

	log.Println("✅ [Router] Started & Listening...")

	<-ctx.Done()

	log.Println("🛑 [Router] Shutting down...")

	return nil
}

func (s *Service) handleEvent(ctx context.Context, event *envelope.Envelope) error {
	log.Printf("🤖 [Router] Received Event ID: %s | Type: %s", event.ID, event.Type)

	// TODO: routing logic here

	return nil
}
