package llm

import "context"

// Role represents the role of the message sender (system, user, assistant, tool).
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Attachment represents a media attachment (image, file).
type Attachment struct {
	Type string `json:"type"` // "image_url", "text_file"
	URL  string `json:"url,omitempty"`
	Data string `json:"data,omitempty"` // Base64 or text content
}

// Message represents a single message in the conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	// Name is optional, used for tool calls to specify which tool is being called or responding.
	Name string `json:"name,omitempty"`
	// ToolCalls is a list of tool calls made by the assistant.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID is the ID of the tool call this message is a response to.
	ToolCallID string `json:"tool_call_id,omitempty"`
	// Attachments is a list of media attachments.
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ToolCall represents a request to call a tool.
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents the function details in a tool call.
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Provider defines the interface for an LLM provider.
type Provider interface {
	// Chat sends a list of messages to the LLM and returns the response.
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*Message, error)
	// Stream sends a list of messages to the LLM and returns a channel of response chunks.
	Stream(ctx context.Context, messages []Message, tools []ToolDefinition) (<-chan string, error)
}

// ToolDefinition represents the schema of a tool that can be passed to the LLM.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the function signature for the LLM.
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters"`
}
