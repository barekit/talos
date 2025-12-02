package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/barekit/talos/pkg/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

type Provider struct {
	client *openai.Client
	model  string
}

func New(opts ...option.RequestOption) *Provider {
	client := openai.NewClient(opts...)
	return &Provider{
		client: &client,
		model:  openai.ChatModelGPT4o, // Default to GPT-4o
	}
}

// SetModel sets the model to use.
func (p *Provider) SetModel(model string) {
	p.model = model
}

func (p *Provider) Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (*llm.Message, error) {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			openaiMessages[i] = openai.SystemMessage(msg.Content)
		case llm.RoleUser:
			if len(msg.Attachments) > 0 {
				parts := []openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(msg.Content),
				}
				for _, att := range msg.Attachments {
					if att.Type == "image_url" {
						parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: att.URL,
						}))
					}
					// Add other types if needed
				}
				openaiMessages[i] = openai.UserMessage(parts)
			} else {
				openaiMessages[i] = openai.UserMessage(msg.Content)
			}
		case llm.RoleAssistant:
			assistantMsg := openai.AssistantMessage(msg.Content)
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					toolCalls[j] = openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
				if assistantMsg.OfAssistant != nil {
					assistantMsg.OfAssistant.ToolCalls = toolCalls
				}
			}
			openaiMessages[i] = assistantMsg
		case llm.RoleTool:
			openaiMessages[i] = openai.ToolMessage(msg.ToolCallID, msg.Content)
		default:
			return nil, fmt.Errorf("unknown role: %s", msg.Role)
		}
	}

	var openaiTools []openai.ChatCompletionToolParam
	if len(tools) > 0 {
		openaiTools = make([]openai.ChatCompletionToolParam, len(tools))
		for i, t := range tools {
			params, ok := t.Function.Parameters.(map[string]interface{})
			if !ok {
				b, _ := json.Marshal(t.Function.Parameters)
				_ = json.Unmarshal(b, &params)
			}

			openaiTools[i] = openai.ChatCompletionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Function.Name,
					Description: openai.String(t.Function.Description),
					Parameters:  shared.FunctionParameters(params),
				},
			}
		}
	}

	params := openai.ChatCompletionNewParams{
		Messages: openaiMessages,
		Model:    p.model,
	}

	if len(openaiTools) > 0 {
		params.Tools = openaiTools
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	choice := completion.Choices[0]
	responseMsg := &llm.Message{
		Role:    llm.RoleAssistant,
		Content: choice.Message.Content,
	}

	if len(choice.Message.ToolCalls) > 0 {
		responseMsg.ToolCalls = make([]llm.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			responseMsg.ToolCalls[i] = llm.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: llm.Function{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return responseMsg, nil
}

// Stream sends a list of messages to the LLM and returns a channel of response chunks.
func (p *Provider) Stream(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (<-chan string, error) {
	// Reuse logic to construct messages (simplified for brevity, ideally refactor)
	// For now, copy-paste the message construction logic or extract it.
	// Extracting message construction logic is better.

	openaiMessages, err := p.buildMessages(messages)
	if err != nil {
		return nil, err
	}

	// Tools are not typically supported in streaming in the same way, or at least we focus on content streaming here.
	// If tools are needed in stream, it gets complex. For now, assume streaming is for text response.

	params := openai.ChatCompletionNewParams{
		Messages: openaiMessages,
		Model:    p.model,
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	out := make(chan string)
	go func() {
		defer close(out)
		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				out <- chunk.Choices[0].Delta.Content
			}
		}
		if err := stream.Err(); err != nil {
			// Handle error? Log it or send it to channel if channel supported errors
			fmt.Printf("Stream error: %v\n", err)
		}
	}()

	return out, nil
}

func (p *Provider) buildMessages(messages []llm.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case llm.RoleSystem:
			openaiMessages[i] = openai.SystemMessage(msg.Content)
		case llm.RoleUser:
			if len(msg.Attachments) > 0 {
				parts := []openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(msg.Content),
				}
				for _, att := range msg.Attachments {
					if att.Type == "image_url" {
						parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: att.URL,
						}))
					}
				}
				openaiMessages[i] = openai.UserMessage(parts)
			} else {
				openaiMessages[i] = openai.UserMessage(msg.Content)
			}
		case llm.RoleAssistant:
			assistantMsg := openai.AssistantMessage(msg.Content)
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					toolCalls[j] = openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
				if assistantMsg.OfAssistant != nil {
					assistantMsg.OfAssistant.ToolCalls = toolCalls
				}
			}
			openaiMessages[i] = assistantMsg
		case llm.RoleTool:
			openaiMessages[i] = openai.ToolMessage(msg.ToolCallID, msg.Content)
		default:
			return nil, fmt.Errorf("unknown role: %s", msg.Role)
		}
	}
	return openaiMessages, nil
}
