package router

import (
	"context"
	"encoding/json"
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
	err := s.bus.Subscribe("data.raw", s.handleRawData, "router_group")
	if err != nil {
		return err
	}

	log.Println("✅ [Router] Subscribed to 'data.raw'")
	<-ctx.Done()
	return nil
}

func (s *Service) handleRawData(ctx context.Context, event *envelope.Envelope) error {
	log.Printf("⚡ [Router] Received Event ID: %s | Type: %s | Tenant: %s", event.ID, event.Type, event.TenantID)

	var data map[string]string
	if err := json.Unmarshal(event.Payload, &data); err != nil {
		log.Printf("   Error decoding payload: %v", err)
		return err
	}

	log.Printf("   Content: %s", data["content"])

	return nil
}
