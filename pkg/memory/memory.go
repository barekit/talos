package memory

import (
	"context"

	"github.com/barekit/talos/pkg/llm"
)

// Memory represents a storage for chat history.
type Memory interface {
	// Save saves a message to the memory for a given session.
	Save(ctx context.Context, sessionID string, msg llm.Message) error
	// Load loads the chat history for a given session.
	Load(ctx context.Context, sessionID string) ([]llm.Message, error)
}
