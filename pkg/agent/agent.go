package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/barekit/talos/pkg/knowledge"
	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/memory"
	"github.com/barekit/talos/pkg/tools"
)

// Agent represents an AI agent.
type Agent struct {
	Name         string
	Instructions string
	LLM          llm.Provider
	Tools        map[string]*tools.Tool
	History      []llm.Message
	MaxSteps     int
	Memory       memory.Memory
	SessionID    string
	Knowledge    *knowledge.KnowledgeBase
	Debug        bool
}

// Option is a function that configures an Agent.
type Option func(*Agent)

// New creates a new Agent.
func New(llmProvider llm.Provider, opts ...Option) *Agent {
	a := &Agent{
		Name:     "Agent",
		LLM:      llmProvider,
		Tools:    make(map[string]*tools.Tool),
		MaxSteps: 10,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// WithName sets the agent's name.
func WithName(name string) Option {
	return func(a *Agent) {
		a.Name = name
	}
}

// WithKnowledge sets the knowledge base for the agent.
func WithKnowledge(kb *knowledge.KnowledgeBase) Option {
	return func(a *Agent) {
		a.Knowledge = kb
	}
}

// WithInstructions sets the agent's system instructions.
func WithInstructions(instructions string) Option {
	return func(a *Agent) {
		a.Instructions = instructions
	}
}

// WithTools adds tools to the agent.
func WithTools(tools ...*tools.Tool) Option {
	return func(a *Agent) {
		for _, t := range tools {
			a.Tools[t.Name] = t
		}
	}
}

// WithMemory sets the agent's memory and session ID.
func WithMemory(mem memory.Memory, sessionID string) Option {
	return func(a *Agent) {
		a.Memory = mem
		a.SessionID = sessionID
	}
}

// WithDebug enables debug logging.
func WithDebug(enable bool) Option {
	return func(a *Agent) {
		a.Debug = enable
	}
}

// Run executes the agent loop with the given input.
func (a *Agent) Run(ctx context.Context, input string, attachments []llm.Attachment) (string, error) {
	if a.Debug {
		slog.Info("Agent Run started", "input", input, "session_id", a.SessionID)
	}

	if err := a.prepareStep(ctx, input, attachments); err != nil {
		if a.Debug {
			slog.Error("Agent Run failed to prepare step", "error", err)
		}
		return "", err
	}

	// Prepare tool definitions
	var toolDefs []llm.ToolDefinition
	for _, t := range a.Tools {
		toolDefs = append(toolDefs, t.Definition)
	}

	steps := 0
	for steps < a.MaxSteps {
		steps++

		if a.Debug {
			slog.Info("Agent Step", "step", steps)
		}

		// Think
		response, err := a.LLM.Chat(ctx, a.History, toolDefs)
		if err != nil {
			if a.Debug {
				slog.Error("LLM Chat failed", "error", err)
			}
			return "", fmt.Errorf("LLM error: %w", err)
		}

		a.History = append(a.History, *response)
		if a.Memory != nil && a.SessionID != "" {
			if err := a.Memory.Save(ctx, a.SessionID, *response); err != nil {
				if a.Debug {
					slog.Error("failed to save assistant message", "error", err)
				}
				return "", fmt.Errorf("failed to save assistant message: %w", err)
			}
		}

		// If no tool calls, we are done
		if len(response.ToolCalls) == 0 {
			if a.Debug {
				slog.Info("Agent Run completed", "response", response.Content)
			}
			return response.Content, nil
		}

		// Act
		for _, tc := range response.ToolCalls {
			if a.Debug {
				slog.Info("Agent Tool Call", "tool", tc.Function.Name, "args", tc.Function.Arguments)
			}

			tool, ok := a.Tools[tc.Function.Name]
			if !ok {
				// Handle unknown tool? For now just error or skip
				// Ideally we tell the LLM the tool is missing
				resultMsg := llm.Message{
					Role:       llm.RoleTool,
					Content:    fmt.Sprintf("Error: Tool %s not found", tc.Function.Name),
					ToolCallID: tc.ID,
				}
				a.History = append(a.History, resultMsg)
				if a.Memory != nil && a.SessionID != "" {
					_ = a.Memory.Save(ctx, a.SessionID, resultMsg)
				}
				continue
			}

			// Execute tool
			output, err := tool.Call(tc.Function.Arguments)
			if err != nil {
				output = fmt.Sprintf("Error executing tool: %v", err)
				if a.Debug {
					slog.Error("Tool execution failed", "tool", tc.Function.Name, "error", err)
				}
			} else {
				if a.Debug {
					slog.Info("Tool execution successful", "tool", tc.Function.Name, "output", output)
				}
			}

			// Observe
			resultMsg := llm.Message{
				Role:       llm.RoleTool,
				Content:    output,
				ToolCallID: tc.ID,
			}
			a.History = append(a.History, resultMsg)
			if a.Memory != nil && a.SessionID != "" {
				if err := a.Memory.Save(ctx, a.SessionID, resultMsg); err != nil {
					if a.Debug {
						slog.Error("failed to save tool output", "error", err)
					}
					return "", fmt.Errorf("failed to save tool output: %w", err)
				}
			}
		}
	}

	if a.Debug {
		slog.Error("Agent Run max steps reached")
	}
	return "", fmt.Errorf("max steps reached")
}

// RunStream executes the agent loop and returns a stream of response chunks.
// Note: Currently supports single-turn streaming (no tool execution loop).
func (a *Agent) RunStream(ctx context.Context, input string, attachments []llm.Attachment) (<-chan string, error) {
	if a.Debug {
		slog.Info("Agent RunStream started", "input", input, "session_id", a.SessionID)
	}

	if err := a.prepareStep(ctx, input, attachments); err != nil {
		if a.Debug {
			slog.Error("Agent RunStream failed to prepare step", "error", err)
		}
		return nil, err
	}

	// Stream from LLM
	// Note: We are not passing tools here to avoid tool call complexity in stream for now
	stream, err := a.LLM.Stream(ctx, a.History, nil)
	if err != nil {
		if a.Debug {
			slog.Error("LLM Stream failed", "error", err)
		}
		return nil, err
	}

	out := make(chan string)
	go func() {
		defer close(out)
		var fullResponse string
		for chunk := range stream {
			fullResponse += chunk
			out <- chunk
		}

		if a.Debug {
			slog.Info("Agent RunStream completed", "response_length", len(fullResponse))
		}

		// Save assistant response to history and memory
		assistantMsg := llm.Message{
			Role:    llm.RoleAssistant,
			Content: fullResponse,
		}
		a.History = append(a.History, assistantMsg)
		if a.Memory != nil && a.SessionID != "" {
			if err := a.Memory.Save(ctx, a.SessionID, assistantMsg); err != nil && a.Debug {
				slog.Error("failed to save assistant message", "error", err)
			}
		}
	}()

	return out, nil
}

// prepareStep handles common logic for preparing the agent step:
// loading history, retrieving RAG context, and saving user input.
func (a *Agent) prepareStep(ctx context.Context, input string, attachments []llm.Attachment) error {
	// Load history from memory if available
	if a.Memory != nil && a.SessionID != "" {
		history, err := a.Memory.Load(ctx, a.SessionID)
		if err != nil {
			return fmt.Errorf("failed to load memory: %w", err)
		}
		a.History = history
	}

	// Initialize history with system prompt if empty
	if len(a.History) == 0 && a.Instructions != "" {
		sysMsg := llm.Message{
			Role:    llm.RoleSystem,
			Content: a.Instructions,
		}
		a.History = append(a.History, sysMsg)
		if a.Memory != nil && a.SessionID != "" {
			_ = a.Memory.Save(ctx, a.SessionID, sysMsg)
		}
	}

	// RAG: Retrieve relevant documents if Knowledge is set
	var contextInfo string
	if a.Knowledge != nil {
		docs, err := a.Knowledge.Retrieve(ctx, input, 3)
		if err != nil {
			return fmt.Errorf("failed to retrieve documents: %w", err)
		}
		if len(docs) > 0 {
			contextInfo = "\nRelevant Context:\n"
			for _, doc := range docs {
				contextInfo += fmt.Sprintf("- %s\n", doc.Content)
			}
		}
	}

	fullInput := input
	if contextInfo != "" {
		fullInput += contextInfo
	}

	userMsg := llm.Message{
		Role:        llm.RoleUser,
		Content:     fullInput,
		Attachments: attachments,
	}
	a.History = append(a.History, userMsg)
	if a.Memory != nil && a.SessionID != "" {
		if err := a.Memory.Save(ctx, a.SessionID, userMsg); err != nil {
			return fmt.Errorf("failed to save user message: %w", err)
		}
	}

	return nil
}
