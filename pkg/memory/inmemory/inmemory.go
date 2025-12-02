package inmemory

import (
	"context"
	"sync"

	"github.com/barekit/talos/pkg/llm"
)

// InMemory implements Memory using a map.
type InMemory struct {
	mu       sync.RWMutex
	messages map[string][]llm.Message
}

// New creates a new InMemory adapter.
func New() *InMemory {
	return &InMemory{
		messages: make(map[string][]llm.Message),
	}
}

// Save saves a message to the in-memory store.
func (m *InMemory) Save(ctx context.Context, sessionID string, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

// Load loads messages from the in-memory store.
func (m *InMemory) Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions if the caller modifies the slice
	msgs := m.messages[sessionID]
	result := make([]llm.Message, len(msgs))
	copy(result, msgs)

	return result, nil
}
