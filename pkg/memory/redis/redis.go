package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/barekit/talos/pkg/llm"
	"github.com/redis/go-redis/v9"
)

// RedisMemory implements Memory using Redis.
type RedisMemory struct {
	client *redis.Client
}

// New creates a new RedisMemory.
func New(client *redis.Client) *RedisMemory {
	return &RedisMemory{client: client}
}

// Save saves a message to Redis.
// Messages are stored as a JSON list under "session:{sessionID}".
func (m *RedisMemory) Save(ctx context.Context, sessionID string, msg llm.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("session:%s", sessionID)
	return m.client.RPush(ctx, key, b).Err()
}

// Load loads messages from Redis.
func (m *RedisMemory) Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	// Get all items in the list
	result, err := m.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]llm.Message, len(result))
	for i, item := range result {
		var msg llm.Message
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message at index %d: %w", i, err)
		}
		messages[i] = msg
	}

	return messages, nil
}
