package engine

import (
	"context"

	"github.com/shanth1/morphic-monad/internal/infra/pkg/domain"
	"github.com/shanth1/morphic-monad/pkg/events"
)

// EventSubscriber - the port for NATS listening
type EventSubscriber interface {
	Subscribe(ctx context.Context, topic events.Topic, queueGroup string, handler events.Handler) error
}

// EventPublisher - port for sending a response by Gateway
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}

// VectorDB is an infrastructure port for working with vector storage.
type VectorDB interface {
	Upsert(ctx context.Context, tenantID domain.TenantID, docID domain.DocumentID, chunkID domain.ChunkID, vector domain.Vector) error
	Search(ctx context.Context, tenantID domain.TenantID, queryVector domain.Vector, topK int) ([]SearchResult, error)
}
