package embedder

import (
	"context"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type Service struct {
	subscriber EventSubscriber
	publisher  EventPublisher
	blobReader BlobReader
	vectoriser TextVectoriser
	logger     log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, blob BlobReader, vec TextVectoriser, l log.Logger) *Service {
	return &Service{subscriber: sub, publisher: pub, blobReader: blob, vectoriser: vec, logger: l}
}

func (s *Service) Start(ctx context.Context) error {
	_ = s.subscriber.Subscribe(ctx, events.TopicTaskEmbed, events.QueueGroupEmbedder, s.handleTask)
	s.logger.Info().Msg("embedder worker started")
	<-ctx.Done()
	return nil
}

func (s *Service) handleTask(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	var docID, chunkID, textToEmbed, blobURI, mimeType string
	var origType events.EventType
	var searchTopK int

	if env.Type == events.EventIngestRequested {
		var p events.IngestPayload
		_ = env.DecodeData(&p)
		docID, chunkID, textToEmbed, blobURI, mimeType, origType = p.DocumentID, p.DocumentID, p.ContextText, p.BlobURI, p.MimeType, env.Type
	} else if env.Type == events.EventSearchRequested {
		var p events.SearchPayload
		_ = env.DecodeData(&p)
		docID, chunkID, textToEmbed, blobURI, mimeType, origType = "search", "search", p.QueryText, p.BlobURI, p.MimeType, env.Type
		searchTopK = p.TopK
	} else if env.Type == events.EventTaskVisionCompleted || env.Type == events.EventTaskChunkCompleted {
		var p events.EmbedRequestedPayload
		_ = env.DecodeData(&p)
		docID, chunkID, textToEmbed, blobURI, mimeType, origType = p.DocumentID, p.ChunkID, p.Text, p.BlobURI, p.MimeType, p.OriginalType
	} else {
		return msg.Ack()
	}

	// Вызов адаптера через интерфейс (чистая бизнес-логика)
	vector, err := s.vectoriser.Vectorise(ctx, textToEmbed)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to vectorise text")
		return msg.Nack()
	}

	if len(vector) == 0 {
		s.logger.Warn().Msg("received empty vector")
		return msg.Ack()
	}

	chunks := []events.Chunk{{
		ChunkID:  chunkID,
		Text:     textToEmbed,
		Vector:   vector,
		BlobURI:  blobURI,
		MimeType: mimeType,
	}}

	resultPayload := events.EmbedCompletedPayload{
		DocumentID:   docID,
		Chunks:       chunks,
		OriginalType: origType,
		SearchTopK:   searchTopK,
	}

	resultEnv, _ := events.NewEnvelope(env.TenantID, env.CorrelationID, events.EventTaskEmbedCompleted, "worker.embedder", resultPayload)
	_ = s.publisher.Publish(ctx, events.TopicIngress, resultEnv)

	s.logger.Info().Str("doc_id", docID).Msg("vectorisation successful")
	return msg.Ack()
}
