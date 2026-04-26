package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/infra/metrics"
	"github.com/shanth1/morphic-monad/internal/infra/pkg/domain"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type Service struct {
	subscriber EventSubscriber
	publisher  EventPublisher
	vectorDB   VectorDB
	logger     log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, db VectorDB, l log.Logger) *Service {
	return &Service{
		subscriber: sub,
		publisher:  pub,
		vectorDB:   db,
		logger:     l,
	}
}

func (s *Service) Start(ctx context.Context) error {
	err := s.subscriber.Subscribe(ctx, events.TopicTaskEngine, events.QueueGroupEngine, s.handleTask)
	if err != nil {
		return fmt.Errorf("engine subscribe: %w", err)
	}

	s.logger.Info().Msg("engine module started, listening for vectorized tasks")
	<-ctx.Done()
	s.logger.Info().Msg("engine module shutting down")
	return nil
}

func (s *Service) handleTask(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	// Engine only responds to completed vectorization tasks
	if env.Type != events.EventTaskEmbedCompleted {
		s.logger.Warn().Str("type", string(env.Type)).Msg("unsupported event type received in engine")
		return msg.Ack()
	}

	var payload events.EmbedCompletedPayload
	if err := env.DecodeData(&payload); err != nil {
		s.logger.Error().Err(err).Msg("failed to decode embed payload")
		return msg.Term()
	}

	tenantID := domain.TenantID(env.TenantID)

	// Original Intent Based Routing (Context Passing Pattern)
	switch payload.OriginalType {
	case events.EventIngestRequested:
		return s.handleUpsert(ctx, tenantID, payload, msg)
	case events.EventSearchRequested:
		return s.handleSearch(ctx, tenantID, env.CorrelationID, payload, msg)
	default:
		s.logger.Warn().Str("original_type", string(payload.OriginalType)).Msg("unknown original type")
		return msg.Ack()
	}
}

// handleUpsert saves all document chunks to the VectorDB
func (s *Service) handleUpsert(ctx context.Context, tenantID domain.TenantID, payload events.EmbedCompletedPayload, msg events.Message) error {
	start := time.Now()
	docID := domain.DocumentID(payload.DocumentID)

	for _, chunk := range payload.Chunks {
		chunkID := domain.ChunkID(chunk.ChunkID)
		vector := domain.Vector(chunk.Vector)

		err := s.vectorDB.Upsert(ctx, tenantID, docID, chunkID, vector)
		if err != nil {
			s.logger.Error().Err(err).Str("doc_id", string(docID)).Msg("failed to upsert vector")
			return msg.Nack() // Return to queue for retry
		}
	}

	duration := time.Since(start).Seconds()
	metrics.WorkerProcessingTime.WithLabelValues("engine", "upsert").Observe(duration)
	metrics.VectorsUpsertedTotal.Add(float64(len(payload.Chunks)))

	s.logger.Info().Str("doc_id", string(docID)).Int("chunks", len(payload.Chunks)).Msg("document vectors saved to DB")
	return msg.Ack()
}

// handleSearch finds nearest vectors and sends results back to the request initiator
func (s *Service) handleSearch(ctx context.Context, tenantID domain.TenantID, correlationID string, payload events.EmbedCompletedPayload, msg events.Message) error {
	start := time.Now()

	if len(payload.Chunks) == 0 {
		s.logger.Warn().Msg("search payload contains no vectors")
		return msg.Term()
	}

	// Search queries typically have 1 chunk (the vectorized query itself)
	queryVector := domain.Vector(payload.Chunks[0].Vector)
	topK := payload.SearchTopK
	if topK <= 0 {
		topK = 5 // Default TopK
	}

	// Perform strict multitenant search in DB
	searchResults, err := s.vectorDB.Search(ctx, tenantID, queryVector, topK)
	if err != nil {
		s.logger.Error().Err(err).Msg("vector search failed")
		return msg.Nack()
	}

	// Convert domain types back to transport primitives
	var finalResults []events.SearchResult
	for _, res := range searchResults {
		finalResults = append(finalResults, events.SearchResult{
			DocumentID: string(res.DocumentID),
			ChunkID:    string(res.ChunkID),
			Score:      res.Score,
		})
	}

	resultPayload := events.SearchCompletedPayload{
		Results: finalResults,
	}

	// Create a response envelope
	resultEnv, err := events.NewEnvelope(string(tenantID), correlationID, events.EventSearchCompleted, "engine", resultPayload)
	if err != nil {
		return msg.Term()
	}

	// Request-Reply Pattern: send result to a dynamic reply topic
	replyTopic := events.Topic(fmt.Sprintf("platform.replies.%s", correlationID))
	if err := s.publisher.Publish(ctx, replyTopic, resultEnv); err != nil {
		s.logger.Error().Err(err).Msg("failed to publish search results")
		return msg.Nack()
	}

	duration := time.Since(start).Seconds()
	metrics.WorkerProcessingTime.WithLabelValues("engine", "search").Observe(duration)

	s.logger.Info().Str("correlation_id", correlationID).Int("hits", len(finalResults)).Msg("search completed successfully")
	return msg.Ack()
}
