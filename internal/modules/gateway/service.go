package gateway

import (
	"context"
	"log"
	"time"

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
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("🚀 [Gateway] Started. Sending heartbeat events...")

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
				log.Printf("Error creating envelope: %v", err)
				continue
			}

			log.Printf("📢 [Gateway] Publishing Event ID: %s", event.ID)
			if err := s.bus.Publish(ctx, "data.raw", event); err != nil {
				log.Printf("❌ Gateway Publish Error: %v", err)
			}
		}
	}
}
