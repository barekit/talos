package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/tools"
)

type mockProvider struct {
	responses      []llm.Message
	callCount      int
	streamResponse string
	err            error
}

func (m *mockProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (*llm.Message, error) {
	if m.callCount >= len(m.responses) {
		return &llm.Message{Role: llm.RoleAssistant, Content: "No more responses"}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *mockProvider) Stream(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (<-chan string, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan string, 1)
	ch <- m.streamResponse
	close(ch)
	return ch, nil
}

type CalculatorArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

func Add(args CalculatorArgs) (string, error) {
	return fmt.Sprintf("%d", args.A+args.B), nil
}

func TestAgent_Run(t *testing.T) {
	// Mock LLM responses
	mock := &mockProvider{
		responses: []llm.Message{
			{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.Function{
							Name:      "Add",
							Arguments: `{"a": 2, "b": 2}`,
						},
					},
				},
			},
			{
				Role:    llm.RoleAssistant,
				Content: "The answer is 4",
			},
		},
	}

	addTool, err := tools.New("Add", "Adds two numbers", Add)
	if err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	a := agent.New(mock, agent.WithTools(addTool))

	ctx := context.Background()
	// Run agent
	response, err := a.Run(ctx, "Calculate 5 + 3", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if response != "The answer is 4" {
		t.Errorf("Expected 'The answer is 4', got '%s'", response)
	}
}
