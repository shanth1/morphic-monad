package engine

import "github.com/shanth1/morphic-monad/internal/infra/pkg/domain"

// SearchResult represents one document found.
type SearchResult struct {
	DocumentID domain.DocumentID
	ChunkID    domain.ChunkID
	Score      float32 // From 0.0 to 1.0 (the closer to 1, the more similar)
}
