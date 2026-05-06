package embedder

import (
	"context"
	"io"

	"github.com/shanth1/morphic-monad/pkg/events"
)

// EventSubscriber - port for receiving tasks
type EventSubscriber interface {
	Subscribe(ctx context.Context, topic events.Topic, queueGroup string, handler events.Handler) error
}

// EventPublisher - a port for sending work results back to the bus
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}

// BlobReader is a port for downloading large files by their URI (Claim Check)
type BlobReader interface {
	Download(ctx context.Context, uri string) (io.ReadCloser, error)
}

type TextVectoriser interface {
	Vectorise(ctx context.Context, text string) ([]float32, error)
}
