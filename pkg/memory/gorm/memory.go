package gorm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/memory/consts"
	"gorm.io/gorm"
)

// Memory implements Memory using GORM.
type Memory struct {
	db *gorm.DB
}

// MessageModel represents the database schema for a message.
type MessageModel struct {
	gorm.Model
	SessionID  string `gorm:"index"`
	Role       string
	Content    string
	ToolCalls  []byte `gorm:"type:json"` // Store as JSON bytes
	ToolCallID string
}

// TableName overrides the table name.
func (MessageModel) TableName() string {
	return consts.TableNameMessages
}

// New creates a new Memory.
func New(db *gorm.DB) (*Memory, error) {
	if err := db.AutoMigrate(&MessageModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}
	return &Memory{db: db}, nil
}

// Save saves a message to the database.
func (m *Memory) Save(ctx context.Context, sessionID string, msg llm.Message) error {
	var toolCallsJSON []byte
	if len(msg.ToolCalls) > 0 {
		b, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsJSON = b
	}

	model := MessageModel{
		SessionID:  sessionID,
		Role:       string(msg.Role),
		Content:    msg.Content,
		ToolCalls:  toolCallsJSON,
		ToolCallID: msg.ToolCallID,
	}

	return m.db.WithContext(ctx).Create(&model).Error
}

// Load loads messages from the database.
func (m *Memory) Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	var models []MessageModel
	if err := m.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&models).Error; err != nil {
		return nil, err
	}

	messages := make([]llm.Message, len(models))
	for i, model := range models {
		msg := llm.Message{
			Role:       llm.Role(model.Role),
			Content:    model.Content,
			ToolCallID: model.ToolCallID,
		}

		if len(model.ToolCalls) > 0 {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal(model.ToolCalls, &toolCalls); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool calls for msg %d: %w", model.ID, err)
			}
			msg.ToolCalls = toolCalls
		}

		messages[i] = msg
	}

	return messages, nil
}
