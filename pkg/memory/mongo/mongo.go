package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/memory/consts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoMemory struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type MessageDoc struct {
	SessionID  string    `bson:"session_id"`
	Role       string    `bson:"role"`
	Content    string    `bson:"content"`
	ToolCalls  string    `bson:"tool_calls,omitempty"` // Stored as JSON string
	ToolCallID string    `bson:"tool_call_id,omitempty"`
	CreatedAt  time.Time `bson:"created_at"`
}

// New creates a new MongoMemory adapter.
func New(client *mongo.Client, dbName, collectionName string) *MongoMemory {
	return &MongoMemory{
		client:     client,
		collection: client.Database(dbName).Collection(collectionName),
	}
}

func (m *MongoMemory) Save(ctx context.Context, sessionID string, msg llm.Message) error {
	var toolCallsJSON string
	if len(msg.ToolCalls) > 0 {
		b, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsJSON = string(b)
	}

	doc := MessageDoc{
		SessionID:  sessionID,
		Role:       string(msg.Role),
		Content:    msg.Content,
		ToolCalls:  toolCallsJSON,
		ToolCallID: msg.ToolCallID,
		CreatedAt:  time.Now(),
	}

	_, err := m.collection.InsertOne(ctx, doc)
	return err
}

func (m *MongoMemory) Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	filter := bson.M{consts.ColSessionID: sessionID}
	opts := options.Find().SetSort(bson.M{consts.ColCreatedAt: 1})

	cursor, err := m.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []llm.Message
	for cursor.Next(ctx) {
		var doc MessageDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		msg := llm.Message{
			Role:       llm.Role(doc.Role),
			Content:    doc.Content,
			ToolCallID: doc.ToolCallID,
		}

		if doc.ToolCalls != "" {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal([]byte(doc.ToolCalls), &toolCalls); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool calls: %w", err)
			}
			msg.ToolCalls = toolCalls
		}

		messages = append(messages, msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}
