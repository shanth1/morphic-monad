package vision

import (
	"context"
	"io"

	"github.com/shanth1/morphic-monad/pkg/events"
)

type EventSubscriber interface {
	Subscribe(ctx context.Context, topic events.Topic, queueGroup string, handler events.Handler) error
}
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}
type BlobReader interface {
	Download(ctx context.Context, uri string) (io.ReadCloser, error)
}

type ImageDescriber interface {
	Describe(ctx context.Context, base64Image string) (string, error)
}
