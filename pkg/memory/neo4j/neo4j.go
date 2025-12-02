package neo4j

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/memory/consts"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jMemory struct {
	driver neo4j.DriverWithContext
	dbName string
}

// New creates a new Neo4jMemory adapter.
func New(uri, username, password, dbName string) (*Neo4jMemory, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, err
	}

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		return nil, err
	}

	return &Neo4jMemory{
		driver: driver,
		dbName: dbName,
	}, nil
}

func (m *Neo4jMemory) Save(ctx context.Context, sessionID string, msg llm.Message) error {
	session := m.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.dbName})
	defer session.Close(ctx)

	var toolCallsJSON string
	if len(msg.ToolCalls) > 0 {
		b, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool calls: %w", err)
		}
		toolCallsJSON = string(b)
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create Session node if not exists
		querySession := fmt.Sprintf(`
		MERGE (s:%s {id: $sessionID})
		RETURN s
		`, consts.LabelSession)
		if _, err := tx.Run(ctx, querySession, map[string]any{"sessionID": sessionID}); err != nil {
			return nil, err
		}

		// Create Message node and link to Session
		queryMsg := fmt.Sprintf(`
		MATCH (s:%s {id: $sessionID})
		CREATE (m:%s {
			%s: $role,
			%s: $content,
			%s: $toolCalls,
			%s: $toolCallID,
			%s: datetime()
		})
		CREATE (s)-[:%s]->(m)
		RETURN m
		`, consts.LabelSession, consts.LabelMessage,
			consts.ColRole, consts.ColContent, consts.ColToolCalls, consts.ColToolCallID, consts.ColCreatedAt,
			consts.RelHasMessage)

		params := map[string]any{
			"sessionID":  sessionID,
			"role":       string(msg.Role),
			"content":    msg.Content,
			"toolCalls":  toolCallsJSON,
			"toolCallID": msg.ToolCallID,
		}
		_, err := tx.Run(ctx, queryMsg, params)
		return nil, err
	})

	return err
}

func (m *Neo4jMemory) Load(ctx context.Context, sessionID string) ([]llm.Message, error) {
	session := m.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: m.dbName})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
		MATCH (s:%s {id: $sessionID})-[:%s]->(m:%s)
		RETURN m.%s, m.%s, m.%s, m.%s
		ORDER BY m.%s ASC
		`, consts.LabelSession, consts.RelHasMessage, consts.LabelMessage,
			consts.ColRole, consts.ColContent, consts.ColToolCalls, consts.ColToolCallID,
			consts.ColCreatedAt)

		result, err := tx.Run(ctx, query, map[string]any{"sessionID": sessionID})
		if err != nil {
			return nil, err
		}

		var messages []llm.Message
		for result.Next(ctx) {
			record := result.Record()

			role, _ := record.Get("m." + consts.ColRole)
			content, _ := record.Get("m." + consts.ColContent)
			toolCallsStr, _ := record.Get("m." + consts.ColToolCalls)
			toolCallID, _ := record.Get("m." + consts.ColToolCallID)

			msg := llm.Message{
				Role:       llm.Role(role.(string)),
				Content:    content.(string),
				ToolCallID: toolCallID.(string),
			}

			if toolCallsStr != nil && toolCallsStr.(string) != "" {
				var toolCalls []llm.ToolCall
				if err := json.Unmarshal([]byte(toolCallsStr.(string)), &toolCalls); err != nil {
					return nil, err
				}
				msg.ToolCalls = toolCalls
			}

			messages = append(messages, msg)
		}

		return messages, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]llm.Message), nil
}

func (m *Neo4jMemory) Close(ctx context.Context) error {
	return m.driver.Close(ctx)
}
