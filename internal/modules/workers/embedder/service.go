package embedder

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"

	"github.com/google/uuid"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/internal/infra/pkg/domain"
	"github.com/shanth1/morphic-monad/pkg/events"
)

const MockVectorDimension = 384

type Service struct {
	subscriber EventSubscriber
	publisher  EventPublisher
	blobReader BlobReader
	logger     log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, blob BlobReader, l log.Logger) *Service {
	return &Service{
		subscriber: sub,
		publisher:  pub,
		blobReader: blob,
		logger:     l,
	}
}

func (s *Service) Start(ctx context.Context) error {
	// Subscribing to tasks for the Embedder.
	// QueueGroup guarantees that if we launch multiple worker instances, only one will process a specific task.
	err := s.subscriber.Subscribe(ctx, events.TopicTaskEmbed, events.QueueGroupEmbedder, s.handleTask)
	if err != nil {
		return fmt.Errorf("embedder subscribe: %w", err)
	}

	s.logger.Info().Msg("embedder worker started, listening for tasks")

	<-ctx.Done()
	s.logger.Info().Msg("embedder worker shutting down")
	return nil
}

func (s *Service) handleTask(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	var textToEmbed string
	var documentID domain.DocumentID

	// 1. Decode payload based on the original event type
	switch env.Type {
	case events.EventIngestRequested:
		var payload events.IngestPayload
		if err := env.DecodeData(&payload); err != nil {
			return msg.Term()
		}
		documentID = domain.DocumentID(payload.DocumentID) // Type casting string -> domain.ID
		textToEmbed = payload.ContextText

		if payload.BlobURI != "" {
			blobData, err := s.readBlobContent(ctx, payload.BlobURI)
			if err != nil {
				s.logger.Error().Err(err).Msg("failed to read blob")
				return msg.Nack()
			}
			textToEmbed += " " + blobData // Early fusion for multimodal embedding
		}

	case events.EventSearchRequested:
		var payload events.SearchPayload
		if err := env.DecodeData(&payload); err != nil {
			return msg.Term()
		}
		textToEmbed = payload.QueryText

		if payload.BlobURI != "" {
			blobData, err := s.readBlobContent(ctx, payload.BlobURI)
			if err != nil {
				return msg.Nack()
			}
			textToEmbed += " " + blobData
		}

	default:
		return msg.Ack()
	}

	// 2. Vectorize content (Mock implementation)
	chunkVector := s.mockVectorize(textToEmbed)
	chunkID := domain.ChunkID(uuid.NewString())

	// 3. Prepare resulting chunks (1-to-1 mapping for MVP, 1-to-N in real scenarios)
	chunks := []events.Chunk{
		{
			ChunkID: string(chunkID), // Type casting domain.ID -> string
			Vector:  chunkVector,
		},
	}

	// 4. Form the response payload, preserving the OriginalType
	resultPayload := events.EmbedCompletedPayload{
		DocumentID:   string(documentID), // Type casting domain.ID -> string
		Chunks:       chunks,
		OriginalType: env.Type,
	}

	resultEnv, err := events.NewEnvelope(env.TenantID, env.CorrelationID, events.EventTaskEmbedCompleted, "worker.embedder", resultPayload)
	if err != nil {
		return msg.Nack()
	}

	// 5. Publish to Ingress (Router will route it to Engine)
	if err := s.publisher.Publish(ctx, events.TopicIngress, resultEnv); err != nil {
		return msg.Nack()
	}

	s.logger.Info().Str("correlation_id", env.CorrelationID).Msg("vectorization completed")
	return msg.Ack()
}

// readBlobContent reads a file from storage
func (s *Service) readBlobContent(ctx context.Context, uri string) (string, error) {
	rc, err := s.blobReader.Download(ctx, uri)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// mockVectorize creates a pseudo-random deterministic vector based on text hash
func (s *Service) mockVectorize(text string) []float32 {
	hash := sha256.Sum256([]byte(text))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	rng := rand.New(rand.NewSource(seed))

	vector := make([]float32, MockVectorDimension)
	for i := 0; i < MockVectorDimension; i++ {
		vector[i] = rng.Float32()
	}
	return vector
}
