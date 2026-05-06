package vision

import (
	"context"
	"encoding/base64"
	"io"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type Service struct {
	sub       EventSubscriber
	pub       EventPublisher
	blob      BlobReader
	describer ImageDescriber
	logger    log.Logger
}

func NewService(sub EventSubscriber, pub EventPublisher, blob BlobReader, desc ImageDescriber, l log.Logger) *Service {
	return &Service{sub: sub, pub: pub, blob: blob, describer: desc, logger: l}
}

func (s *Service) Start(ctx context.Context) error {
	_ = s.sub.Subscribe(ctx, events.TopicTaskVision, events.QueueGroupVision, s.handleTask)
	s.logger.Info().Msg("vision worker started")
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
	} else if env.Type == events.EventSearchRequested {
		var p events.SearchPayload
		_ = env.DecodeData(&p)
		docID, blobURI, mimeType, origType = "search-img", p.BlobURI, p.MimeType, env.Type
	} else {
		return msg.Ack()
	}

	rc, err := s.blob.Download(ctx, blobURI)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read blob")
		return msg.Nack()
	}
	defer rc.Close()
	imgData, _ := io.ReadAll(rc)
	b64Img := base64.StdEncoding.EncodeToString(imgData)

	description, err := s.describer.Describe(ctx, b64Img)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to describe image")
		return msg.Nack()
	}

	outPayload := events.EmbedRequestedPayload{
		DocumentID:   docID,
		ChunkID:      docID + "-vision",
		Text:         description,
		BlobURI:      blobURI,
		MimeType:     mimeType,
		OriginalType: origType,
	}

	outEnv, _ := events.NewEnvelope(env.TenantID, env.CorrelationID, events.EventTaskVisionCompleted, "worker.vision", outPayload)
	_ = s.pub.Publish(ctx, events.TopicIngress, outEnv)

	s.logger.Info().Str("doc_id", docID).Msg("image described successfully")
	return msg.Ack()
}
