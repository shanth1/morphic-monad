package engine

import "github.com/shanth1/morphic-monad/internal/infra/pkg/domain"

// SearchQuery is an internal business model for searching queries within Engine.
type SearchQuery struct {
	QueryText string
	TopK      int
	Filters   map[string]string
}

// Chunk represents a fragment of text to be vectorized.
type Chunk struct {
	ChunkID    domain.DocumentID
	DocumentID domain.DocumentID
	Text       string
	Metadata   map[string]string
}
