package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/knowledge"
	"github.com/barekit/talos/pkg/knowledge/openai"
	postgresvec "github.com/barekit/talos/pkg/knowledge/postgres"
	"github.com/barekit/talos/pkg/knowledge/qdrant"
	llmopenai "github.com/barekit/talos/pkg/llm/openai"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Embedder
	embedder := openai.NewEmbedder()

	// 2. Initialize Vector Store
	var vectorStore knowledge.VectorStore
	var err error

	storeType := os.Getenv("VECTOR_STORE")
	if storeType == "postgres" {
		dsn := os.Getenv("POSTGRES_DSN")
		if dsn == "" {
			dsn = "host=localhost user=postgres password=postgres dbname=talos port=5432 sslmode=disable"
		}
		vectorStore, err = postgresvec.New(dsn)
	} else {
		// Default to Qdrant
		qdrantHost := "localhost"
		qdrantPort := 6334
		if h := os.Getenv("QDRANT_HOST"); h != "" {
			qdrantHost = h
		}
		vectorStore, err = qdrant.New(qdrantHost, qdrantPort, "talos_knowledge", 1536)
	}

	if err != nil {
		log.Fatalf("Failed to initialize vector store: %v", err)
	}

	// 3. Initialize Knowledge Base
	kb := knowledge.NewKnowledgeBase(embedder, vectorStore)

	// 4. Ingest Data (if requested)
	if len(os.Args) > 1 && os.Args[1] == "ingest" {
		docs := []knowledge.Document{
			{
				ID:      "1",
				Content: "Talos is a modular AI agent framework written in Go.",
				Metadata: map[string]interface{}{
					"source": "manual",
				},
			},
			{
				ID:      "2",
				Content: "Talos supports multiple memory backends including SQLite, Postgres, Redis, Neo4j, and MongoDB.",
				Metadata: map[string]interface{}{
					"source": "manual",
				},
			},
			{
				ID:      "3",
				Content: "Phase 3 of Talos development focuses on Knowledge and RAG capabilities.",
				Metadata: map[string]interface{}{
					"source": "manual",
				},
			},
		}

		if err := kb.Ingest(ctx, docs); err != nil {
			log.Fatalf("Failed to ingest documents: %v", err)
		}
		fmt.Println("Ingestion complete.")
		return
	}

	// 5. Initialize Agent with Knowledge
	llmProvider := llmopenai.New()
	myAgent := agent.New(
		llmProvider,
		agent.WithInstructions("You are a helpful assistant. Use the provided context to answer questions."),
		agent.WithKnowledge(kb),
	)

	// 6. Run Agent
	input := "What memory backends does Talos support?"
	if len(os.Args) > 1 {
		input = os.Args[1]
	}

	fmt.Printf("User: %s\n", input)
	response, err := myAgent.Run(ctx, input, nil)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	fmt.Printf("Assistant: %s\n", response)
}
