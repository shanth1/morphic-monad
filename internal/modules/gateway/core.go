package gateway

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/pkg/events"
)

type Service struct {
	publisher EventPublisher
	blobStore BlobStore
	logger    log.Logger
}

func NewService(pub EventPublisher, blob BlobStore, l log.Logger) *Service {
	return &Service{
		publisher: pub,
		blobStore: blob,
		logger:    l,
	}
}

// IngestDocument is a business case for uploading a new document to the platform.
// Called by the Primary adapter (e.g., an HTTP handler).
func (s *Service) IngestDocument(ctx context.Context, tenantID, filename, mimeType string, size int64, fileReader io.Reader) (string, error) {
	// 1. Checking the claim using the sample: saving a heavy file in storage
	blobURI, err := s.blobStore.Upload(ctx, tenantID, filename, fileReader, size)
	if err != nil {
		s.logger.Error().Err(err).Str("tenant", tenantID).Msg("failed to upload blob")
		return "", fmt.Errorf("upload to blob store: %w", err)
	}

	docID := uuid.NewString()

	// 2. Generate a "claim check"
	payload := events.ClaimCheckPayload{
		DocumentID: docID,
		BlobURI:    blobURI,
		MimeType:   mimeType,
		SizeBytes:  size,
		Metadata: map[string]string{
			"original_name": filename,
		},
	}

	// 3. Pack it into a CloudEvents Envelope
	// The CorrelationID is generated here – it will live throughout the pipeline
	correlationID := uuid.NewString()
	env, err := events.NewEnvelope(tenantID, correlationID, events.EventDocumentUploaded, "gateway", payload)
	if err != nil {
		return "", fmt.Errorf("create envelope: %w", err)
	}

	// TODO:
	// In the future, this will be where TraceID will be extracted from the context (OpenTelemetry)
	// env.TraceID = trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	// 4. Send to the router's startup topic
	if err := s.publisher.Publish(ctx, events.TopicIngress, env); err != nil {
		s.logger.Error().Err(err).Str("doc_id", docID).Msg("failed to publish event")
		return "", fmt.Errorf("publish event: %w", err)
	}

	s.logger.Info().
		Str("doc_id", docID).
		Str("correlation_id", correlationID).
		Str("uri", blobURI).
		Msg("document ingested successfully")

	return docID, nil
}
