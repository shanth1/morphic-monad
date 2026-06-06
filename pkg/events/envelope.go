package events

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Envelope is a universal container for all events in the system
type Envelope struct {
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`

	// Routing
	Type   EventType `json:"type"`
	Source string    `json:"source"`

	// Payload
	Data json.RawMessage `json:"data"`

	// Metadata
	TraceID    string    `json:"trace_id,omitempty"` // OpenTelemetry Trace ID
	SpanID     string    `json:"span_id,omitempty"`  // OpenTelemetry Span ID
	RetryCount int       `json:"retry_count"`        // Number of processing attempts (Dead Letter Queue)
	CreatedAt  time.Time `json:"created_at"`
}

// NewEnvelope enforces strict validation for multitenant EDA platforms
func NewEnvelope(tenantID, correlationID string, eventType EventType, source string, data any) (*Envelope, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is strictly required for any event")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	return &Envelope{
		EventID:       uuid.NewString(),
		CorrelationID: correlationID,
		TenantID:      tenantID,
		Type:          eventType,
		Source:        source,
		Data:          payload,
		RetryCount:    0,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// DecodeData unmarshals the raw JSON payload into a specific struct
func (e *Envelope) DecodeData(v any) error {
	return json.Unmarshal(e.Data, v)
}
