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
	ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			event, _ := envelope.New("tenant-1", "input.text", map[string]string{"msg": "Hello Axon!"})

			log.Println("📢 [Gateway] Ingesting new data...")
			if err := s.bus.Publish(context.Background(), "data.raw", event); err != nil {
				log.Printf("Gateway Publish Error: %v", err)
			}
		}
	}
}
