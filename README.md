# Talos

**Talos** is an open-source framework for building autonomous AI agents in Golang. It aims to be the "Gin/Echo" of AI Agents—lightweight, fast, and batteries-included.

While Python dominates the AI prototyping space, Go lacks a framework that balances **simplicity** with **production readiness**. Talos fills this gap by providing flexible tool usage, built-in memory management, and easy RAG integration, all while leveraging Go's concurrency and type safety.

## Features

- **Idiomatic Go**: Built with interfaces, structs, and functional options.
- **Tool Reflection**: Automatically generate OpenAI JSON Schemas from Go functions.
- **Memory Management**: Built-in support for persistent chat history (SQLite, Postgres, MySQL, MSSQL, Redis, Mongo, Neo4j).
- **RAG Integration**: Easy-to-use Knowledge Base with vector store support (Qdrant, PGVector).
- **Streaming**: Native support for streaming LLM responses.
- **Multi-Modal**: Support for image attachments.
- **Structured Logging**: Production-ready logging with `log/slog`.

## Installation

```bash
go get github.com/barekit/talos
```

## Quick Start

Here's a simple agent that can use a calculator tool:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/llm/openai"
	"github.com/barekit/talos/pkg/tools"
)

// Define a tool
func Add(args struct {
	A int `json:"a"`
	B int `json:"b"`
}) (string, error) {
	return fmt.Sprintf("%d", args.A+args.B), nil
}

func main() {
	// 1. Initialize LLM Provider
	apiKey := os.Getenv("OPENAI_API_KEY")
	llm, err := openai.New(apiKey, "gpt-4o-mini")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Create Tool
	addTool, err := tools.New("Add", "Adds two numbers", Add)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create Agent
	myAgent := agent.New(llm, agent.WithTools(addTool))

	// 4. Run
	ctx := context.Background()
	response, err := myAgent.Run(ctx, "What is 5 + 3?", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Agent:", response)
}
```

## Documentation

### Packages

- **`pkg/agent`**: Core agent orchestration logic (`Run`, `RunStream`).
- **`pkg/llm`**: LLM provider interfaces and adapters (OpenAI).
- **`pkg/tools`**: Reflection-based tool creation and execution.
- **`pkg/memory`**: Chat history persistence (SQL, NoSQL, Graph).
- **`pkg/knowledge`**: RAG pipeline (Embeddings, Vector Stores).

### Configuration

The agent can be configured using functional options:

```go
agent.New(llm,
    agent.WithTools(tools...),
    agent.WithMemory(memory),
    agent.WithKnowledge(knowledgeBase),
    agent.WithInstructions("You are a helpful assistant."),
    agent.WithDebug(true), // Enable structured logging
)
```

## Examples

Check out the `examples/` directory for more use cases:

- **[Calculator](examples/calculator)**: Basic tool usage.
- **[RAG Agent](examples/rag_agent)**: Question answering over documents.
- **[Streaming](examples/streaming_agent)**: Streaming responses and image input.
