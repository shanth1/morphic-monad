package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

// --- MOCK ADAPTER ---
type MockVectoriser struct {
	dimensions int
}

func NewMockVectoriser(dimensions int) *MockVectoriser {
	return &MockVectoriser{dimensions: dimensions}
}

func (m *MockVectoriser) Vectorise(ctx context.Context, text string) ([]float32, error) {
	hash := sha256.Sum256([]byte(text))
	seed := int64(binary.BigEndian.Uint64(hash[:8]))
	rng := rand.New(rand.NewSource(seed))

	vector := make([]float32, m.dimensions)
	for i := 0; i < m.dimensions; i++ {
		vector[i] = rng.Float32()
	}
	return vector, nil
}

// --- OLLAMA ADAPTER ---
type OllamaVectoriser struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaVectoriser(baseURL, model string) *OllamaVectoriser {
	return &OllamaVectoriser{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (o *OllamaVectoriser) Vectorise(ctx context.Context, text string) ([]float32, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":  o.model,
		"prompt": text,
	})

	url := fmt.Sprintf("%s/api/embeddings", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return ollamaResp.Embedding, nil
}
