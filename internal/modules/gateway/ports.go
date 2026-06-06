package gateway

import (
	"context"
	"io"

	"github.com/shanth1/morphic-monad/pkg/events"
)

// EventSubscriber - the port for NATS listening
type EventSubscriber interface {
	Subscribe(ctx context.Context, topic events.Topic, queueGroup string, handler events.Handler) error
	SubscribeEphemeral(ctx context.Context, topic events.Topic, handler func(*events.Envelope)) error
}

// EventPublisher is the Driven Port for Gateway
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}

type GatewayService interface {
	IngestDocument(ctx context.Context, tenantID, correlationID, contextText, filename, mimeType string, size int64, fileReader io.Reader) (string, error)
	SearchDocuments(ctx context.Context, tenantID, queryText, filename, mimeType string, size int64, fileReader io.Reader, topK int) ([]events.SearchResult, error)
	GetBlob(ctx context.Context, uri string) (io.ReadCloser, error)
	ListenEvents(ctx context.Context, correlationID string) (<-chan *events.Envelope, error)
}

// BlobStore is an outgoing port for storing heavy files (Claim Check)
type BlobStore interface {
	// Upload saves the data stream and returns a unique URI (e.g. s3://bucket/file.pdf)
	Upload(ctx context.Context, tenantID, filename string, reader io.Reader, size int64) (string, error)
	Download(ctx context.Context, uri string) (io.ReadCloser, error)
}
