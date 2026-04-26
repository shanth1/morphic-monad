package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/morphic-monad/pkg/events"
)

// The maximum text size we allow to pass directly to NATS (e.g., 64 KB).
// Anything larger will automatically be converted to a file.
const MaxDirectTextSizeBytes = 64 * 1024

// Service is the core module for the Gateway.
type Service struct {
	publisher  EventPublisher
	subscriber EventSubscriber
	blobStore  BlobStore
	logger     log.Logger

	// Reply Router: correlates asynchronous NATS replies with synchronous HTTP requests
	pendingReqs map[string]chan *events.Envelope
	mu          sync.RWMutex
}

// NewService creates a new Gateway core service.
func NewService(pub EventPublisher, sub EventSubscriber, blob BlobStore, l log.Logger) *Service {
	return &Service{
		publisher:   pub,
		subscriber:  sub,
		blobStore:   blob,
		logger:      l,
		pendingReqs: make(map[string]chan *events.Envelope),
	}
}

// Start initializes the Reply Router to listen for search results.
func (s *Service) Start(ctx context.Context) error {
	// Subscribe to ALL replies. QueueGroup is empty because every gateway instance
	// needs to receive the reply to check if it holds the pending HTTP request.
	replyTopic := events.Topic("platform.replies.*")
	err := s.subscriber.Subscribe(ctx, replyTopic, "", s.handleReply)
	if err != nil {
		return fmt.Errorf("gateway subscribe to replies: %w", err)
	}

	s.logger.Info().Msg("gateway reply router started")
	<-ctx.Done()
	s.logger.Info().Msg("gateway shutting down")
	return nil
}

// handleReply processes incoming NATS replies and routes them to waiting HTTP goroutines.
func (s *Service) handleReply(ctx context.Context, msg events.Message) error {
	env := msg.Envelope()

	s.mu.RLock()
	ch, exists := s.pendingReqs[env.CorrelationID]
	s.mu.RUnlock()

	if !exists {
		// No one is waiting for this reply on this specific Gateway instance.
		// It either belongs to another instance or the HTTP request timed out.
		return msg.Ack()
	}

	// Send the envelope to the waiting goroutine
	ch <- env
	return msg.Ack()
}

// IngestDocument is a business use case for uploading new data to the platform.
func (s *Service) IngestDocument(ctx context.Context, tenantID, contextText, filename, mimeType string, size int64, fileReader io.Reader) (string, error) {
	var blobURI string
	var err error

	// Adaptive Claim Check for Large Text
	if len(contextText) > MaxDirectTextSizeBytes {
		s.logger.Warn().
			Str("tenant_id", tenantID).
			Int("text_size", len(contextText)).
			Msg("context text is too large, automatically offloading to blob store")

		textReader := bytes.NewReader([]byte(contextText))
		textFilename := fmt.Sprintf("auto_context_%s.txt", uuid.NewString()[:8])

		blobURI, err = s.blobStore.Upload(ctx, tenantID, textFilename, textReader, int64(len(contextText)))
		if err != nil {
			return "", fmt.Errorf("auto-upload context text to blob store: %w", err)
		}

		contextText = ""
		mimeType = "text/plain"
		size = int64(len(contextText))
	}

	// Upload heavy file to Blob Storage (if provided)
	if fileReader != nil && size > 0 {
		blobURI, err = s.blobStore.Upload(ctx, tenantID, filename, fileReader, size)
		if err != nil {
			s.logger.Error().Err(err).Str("tenant", tenantID).Msg("failed to upload blob")
			return "", fmt.Errorf("upload to blob store: %w", err)
		}
	}

	docID := uuid.NewString()

	payload := events.IngestPayload{
		DocumentID:  docID,
		ContextText: contextText,
		BlobURI:     blobURI,
		MimeType:    mimeType,
		SizeBytes:   size,
		Metadata: map[string]string{
			"original_name": filename,
		},
	}

	correlationID := uuid.NewString()
	env, err := events.NewEnvelope(tenantID, correlationID, events.EventIngestRequested, "gateway", payload)
	if err != nil {
		return "", fmt.Errorf("create envelope: %w", err)
	}

	if err := s.publisher.Publish(ctx, events.TopicIngress, env); err != nil {
		return "", fmt.Errorf("publish event: %w", err)
	}

	s.logger.Info().Str("doc_id", docID).Msg("document ingested successfully")
	return docID, nil
}

// SearchDocuments performs a synchronous search over the asynchronous event bus.
func (s *Service) SearchDocuments(ctx context.Context, tenantID, queryText, blobURI string, topK int) ([]events.SearchResult, error) {
	payload := events.SearchPayload{
		QueryText: queryText,
		BlobURI:   blobURI,
		TopK:      topK,
	}

	correlationID := uuid.NewString()
	env, err := events.NewEnvelope(tenantID, correlationID, events.EventSearchRequested, "gateway", payload)
	if err != nil {
		return nil, fmt.Errorf("create envelope: %w", err)
	}

	// 1. Create a channel and register it in the Reply Router
	replyCh := make(chan *events.Envelope, 1)
	s.mu.Lock()
	s.pendingReqs[correlationID] = replyCh
	s.mu.Unlock()

	// Guarantee cleanup when the function exits
	defer func() {
		s.mu.Lock()
		delete(s.pendingReqs, correlationID)
		s.mu.Unlock()
		close(replyCh)
	}()

	// 2. Publish search request to the bus
	if err := s.publisher.Publish(ctx, events.TopicIngress, env); err != nil {
		return nil, fmt.Errorf("publish search event: %w", err)
	}

	// 3. Wait for the reply with a timeout (e.g., 10 seconds)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("search request timeout")

	case replyEnv := <-replyCh:
		// 4. Reply received! Decode and return.
		var resultPayload events.SearchCompletedPayload
		if err := replyEnv.DecodeData(&resultPayload); err != nil {
			return nil, fmt.Errorf("decode search reply: %w", err)
		}
		return resultPayload.Results, nil
	}
}
