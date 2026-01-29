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

func (s *Service) Start() error {
	return s.bus.Subscribe("data.raw", s.handleEvent, "router-workers")
}

func (s *Service) handleEvent(ctx context.Context, event *envelope.Envelope) error {
	log.Printf("🤖 [Router] Received Event ID: %s | Type: %s", event.ID, event.Type)
	log.Println("    -> Routing logic executing...")
	// Тут логика роутинга...
	return nil
}
