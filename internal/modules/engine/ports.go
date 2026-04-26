package engine

import (
	"context"

	"github.com/shanth1/morphic-monad/internal/infra/pkg/domain"
)

// SearchResult represents one document found.
type SearchResult struct {
	DocumentID domain.DocumentID
	Score      float32 // From 0.0 to 1.0 (the closer to 1, the more similar)
}

// VectorDB is an infrastructure port for working with vector storage.
type VectorDB interface {
	Upsert(ctx context.Context, tenantID domain.TenantID, docID domain.DocumentID, vector domain.Vector) error
	Search(ctx context.Context, tenantID domain.TenantID, queryVector domain.Vector, topK int) ([]SearchResult, error)
}
