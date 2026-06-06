package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- MOCK ADAPTER ---
type MockDescriber struct{}

func NewMockDescriber() *MockDescriber {
	return &MockDescriber{}
}

func (m *MockDescriber) Describe(ctx context.Context, base64Image string) (string, error) {
	return "Mock description: An image containing technical architecture diagrams.", nil
}

// --- OLLAMA ADAPTER ---
type OllamaDescriber struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaDescriber(baseURL, model string) *OllamaDescriber {
	return &OllamaDescriber{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (o *OllamaDescriber) Describe(ctx context.Context, base64Image string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":  o.model,
		"prompt": "Describe this image in detail. Focus on text, diagrams, and key entities.",
		"images": []string{base64Image},
		"stream": false,
	})

	url := fmt.Sprintf("%s/api/generate", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}
