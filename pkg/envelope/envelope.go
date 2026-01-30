package envelope

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Metadata map[string]string

type Envelope struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"tenant_id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Meta      Metadata        `json:"meta,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func New(tenantID, eventType string, payload any) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Envelope{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Meta:      make(Metadata),
		Payload:   payloadBytes,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (e *Envelope) Unpack(target any) error {
	return json.Unmarshal(e.Payload, target)
}
