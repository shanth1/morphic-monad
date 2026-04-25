package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

type MemoryStorage struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		store: make(map[string][]byte),
	}
}

func (m *MemoryStorage) Upload(ctx context.Context, tenantID, filename string, reader io.Reader, size int64) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("in-memory upload read: %w", err)
	}

	uri := fmt.Sprintf("memory://%s/%s", tenantID, filename)

	m.mu.Lock()
	m.store[uri] = data
	m.mu.Unlock()

	return uri, nil
}

func (m *MemoryStorage) Download(ctx context.Context, uri string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.store[uri]
	if !exists {
		return nil, fmt.Errorf("blob not found in memory: %s", uri)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}
