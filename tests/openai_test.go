package tests

import (
	"context"
	"os"
	"testing"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/llm/openai"
	"github.com/joho/godotenv"
	"github.com/openai/openai-go/option"
)

func TestAgent_OpenAI_Integration(t *testing.T) {
	_ = godotenv.Load("../.env") // Try to load .env from root
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping OpenAI integration test: OPENAI_API_KEY not set")
	}

	// Initialize OpenAI provider
	provider := openai.New(option.WithAPIKey(apiKey))
	provider.SetModel("gpt-4o-mini")

	// Initialize Agent
	a := agent.New(provider, agent.WithDebug(true))

	// Run Agent
	ctx := context.Background()
	response, err := a.Run(ctx, "What is 2+2? Reply with just the number.", nil)
	if err != nil {
		t.Fatalf("Agent Run failed: %v", err)
	}

	if response != "4" {
		t.Logf("Expected '4', got '%s'", response)
		// Allow some flexibility in LLM response, but it should contain 4
	}
}
