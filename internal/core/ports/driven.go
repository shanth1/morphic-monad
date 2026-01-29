package ports

import (
	"context"

	"github.com/shanth1/morphic-monad/pkg/envelope"
)

type BusHandler func(ctx context.Context, event *envelope.Envelope) error

type Bus interface {
	Publish(ctx context.Context, topic string, event *envelope.Envelope) error
	Subscribe(topic string, handler BusHandler, queueGroup string) error
	Close() error
}
