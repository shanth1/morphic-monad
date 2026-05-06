package chunker

import (
	"context"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type EventSubscriber interface {
	Subscribe(ctx context.Context, topic events.Topic, queueGroup string, handler events.Handler) error
}
type EventPublisher interface {
	Publish(ctx context.Context, topic events.Topic, env *events.Envelope) error
}
type BlobReader interface {
	Download(ctx context.Context, uri string) (io.ReadCloser, error)
}

type Service struct {
	sub    EventSubscriber
	pub    EventPublisher
	blob   BlobReader
	logger log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, blob BlobReader, l log.Logger) *Service {
	return &Service{sub: sub, pub: pub, blob: blob, logger: l}
}

func (s *Service) Start(ctx context.Context) error {
	_ = s.sub.Subscribe(ctx, events.TopicTaskChunk, events.QueueGroupChunker, s.handleTask)
	s.logger.Info().Msg("chunker worker started")
	<-ctx.Done()
	return nil
}

func (s *Service) handleTask(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	var docID, blobURI, mimeType string
	var origType events.EventType

	if env.Type == events.EventIngestRequested {
		var p events.IngestPayload
		_ = env.DecodeData(&p)
		docID, blobURI, mimeType, origType = p.DocumentID, p.BlobURI, p.MimeType, env.Type
	} else {
		return msg.Ack() // Search requests usually aren't chunked heavily, but could be.
	}

	rc, err := s.blob.Download(ctx, blobURI)
	if err != nil {
		return msg.Nack()
	}
	defer rc.Close()
	textData, _ := io.ReadAll(rc)

	// Split Text (Chunk Explosion)
	chunks := strings.Split(string(textData), "\n\n")

	for _, textChunk := range chunks {
		if strings.TrimSpace(textChunk) == "" {
			continue
		}

		outPayload := events.EmbedRequestedPayload{
			DocumentID:   docID,
			ChunkID:      uuid.NewString(),
			Text:         textChunk,
			BlobURI:      blobURI,
			MimeType:     mimeType,
			OriginalType: origType,
		}

		outEnv, _ := events.NewEnvelope(env.TenantID, env.CorrelationID, events.EventTaskChunkCompleted, "worker.chunker", outPayload)
		_ = s.pub.Publish(ctx, events.TopicIngress, outEnv)
	}

	return msg.Ack()
}
