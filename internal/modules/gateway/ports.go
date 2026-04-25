package gateway

import (
	"context"
	"io"

	"github.com/shanth1/morphic-monad/pkg/events"
)

type IngestService interface {
	IngestDocument(ctx context.Context, tenantID, filename, mimeType string, size int64, fileReader io.Reader) (string, error)
}

// EventPublisher is the Driven Port for Gateway
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}

// BlobStore is an outgoing port for storing heavy files (Claim Check)
type BlobStore interface {
	// Upload saves the data stream and returns a unique URI (e.g. s3://bucket/file.pdf)
	Upload(ctx context.Context, tenantID, filename string, reader io.Reader, size int64) (string, error)
}
