package vectordb

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"

	"github.com/shanth1/morphic-monad/internal/infra/pkg/domain"
	"github.com/shanth1/morphic-monad/internal/modules/engine"
)

var (
	ErrEmptyTenant    = errors.New("tenant_id is required")
	ErrEmptyVector    = errors.New("vector cannot be empty")
	ErrVectorMismatch = errors.New("vector dimensions mismatch")
)

type chunkRecord struct {
	DocID    domain.DocumentID
	Vector   domain.Vector
	Text     string
	FileURI  string
	MimeType string
}

type MemoryVectorDB struct {
	mu    sync.RWMutex
	store map[domain.TenantID]map[domain.ChunkID]chunkRecord
}

func NewMemoryVectorDB() *MemoryVectorDB {
	return &MemoryVectorDB{
		store: make(map[domain.TenantID]map[domain.ChunkID]chunkRecord),
	}
}

// Upsert saves or updates a document vector for a specific tenant.
func (db *MemoryVectorDB) Upsert(
	ctx context.Context,
	tenantID domain.TenantID,
	docID domain.DocumentID,
	chunkID domain.ChunkID,
	vector domain.Vector,
	text string,
	fileURI string,
	mimeType string,
) error {
	if tenantID == "" {
		return ErrEmptyTenant
	}
	if len(vector) == 0 {
		return ErrEmptyVector
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.store[tenantID]; !exists {
		db.store[tenantID] = make(map[domain.ChunkID]chunkRecord)
	}

	db.store[tenantID][chunkID] = chunkRecord{
		DocID:    docID,
		Vector:   vector,
		Text:     text,
		FileURI:  fileURI,
		MimeType: mimeType,
	}
	return nil
}

// Search performs a linear search over the vectors of a specific tenant
func (db *MemoryVectorDB) Search(ctx context.Context, tenantID domain.TenantID, queryVector domain.Vector, topK int) ([]engine.SearchResult, error) {
	if tenantID == "" {
		return nil, ErrEmptyTenant
	}
	if len(queryVector) == 0 {
		return nil, ErrEmptyVector
	}

	db.mu.RLock()
	tenantChunks, exists := db.store[tenantID]
	db.mu.RUnlock()

	if !exists || len(tenantChunks) == 0 {
		return []engine.SearchResult{}, nil
	}

	var results []engine.SearchResult

	for chunkID, record := range tenantChunks {
		if len(record.Vector) != len(queryVector) {
			continue
		}

		score := cosineSimilarity(queryVector, record.Vector)
		results = append(results, engine.SearchResult{
			DocumentID: record.DocID,
			ChunkID:    chunkID,
			Score:      score,
			Text:       record.Text,
			FileURI:    record.FileURI,
			MimeType:   record.MimeType,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// cosineSimilarity calculates the cosine distance between two vectors.
// Returns a value from -1.0 to 1.0 (usually converted to 0..1 for search tasks).
func cosineSimilarity(a, b []float32) float32 {
	var dotProduct float64
	var normA float64
	var normB float64

	for i := range len(a) {
		valA := float64(a[i])
		valB := float64(b[i])

		dotProduct += valA * valB
		normA += valA * valA
		normB += valB * valB
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}
