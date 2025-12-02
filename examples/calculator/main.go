package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/llm/openai"
	"github.com/barekit/talos/pkg/memory"
	"github.com/barekit/talos/pkg/tools"
)

type CalculatorArgs struct {
	A int `json:"a" description:"The first number"`
	B int `json:"b" description:"The second number"`
}

func Add(args CalculatorArgs) (string, error) {
	return fmt.Sprintf("%d", args.A+args.B), nil
}

func Subtract(args CalculatorArgs) (string, error) {
	return fmt.Sprintf("%d", args.A-args.B), nil
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Initialize OpenAI provider
	llmProvider := openai.New()

	// Initialize Memory using Factory
	// Default to SQLite, but can be changed via env var MEMORY_TYPE
	// Examples:
	// MEMORY_TYPE=redis MEMORY_CONN=redis://localhost:6379/0
	// MEMORY_TYPE=neo4j MEMORY_CONN=bolt://localhost:7687
	// MEMORY_TYPE=mongo MEMORY_CONN=mongodb://localhost:27017
	// MEMORY_TYPE=mysql MEMORY_CONN="user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	// MEMORY_TYPE=mssql MEMORY_CONN="sqlserver://username:password@localhost:1433?database=dbname"
	// MEMORY_TYPE=inmemory
	memType := memory.TypeSQLite
	connStr := "talos.db"
	username := ""
	password := ""

	if t := os.Getenv("MEMORY_TYPE"); t != "" {
		memType = memory.Type(t)
	}
	if c := os.Getenv("MEMORY_CONN"); c != "" {
		connStr = c
	}
	if u := os.Getenv("MEMORY_USER"); u != "" {
		username = u
	}
	if p := os.Getenv("MEMORY_PASS"); p != "" {
		password = p
	}
	dbName := ""
	if d := os.Getenv("MEMORY_DB"); d != "" {
		dbName = d
	}

	mem, err := memory.NewFactory(context.Background(), memory.Config{
		Type:             memType,
		ConnectionString: connStr,
		Username:         username,
		Password:         password,
		DBName:           dbName,
	})
	if err != nil {
		log.Fatalf("Failed to initialize memory: %v", err)
	}

	// Create tools
	addTool, err := tools.New("Add", "Adds two numbers", Add)
	if err != nil {
		log.Fatalf("Failed to create Add tool: %v", err)
	}

	subTool, err := tools.New("Subtract", "Subtracts two numbers", Subtract)
	if err != nil {
		log.Fatalf("Failed to create Subtract tool: %v", err)
	}

	// Create agent

	// Initialize Agent with memory and debug logging
	myAgent := agent.New(
		llmProvider,
		agent.WithInstructions("You are a calculator agent. Use the 'Add' tool to perform calculations."),
		agent.WithTools(addTool, subTool),
		agent.WithMemory(mem, "session-123"),
		agent.WithDebug(true),
	)

	// Run agent
	ctx := context.Background()
	input := "What is 10 plus 5, and then minus 3?"
	if len(os.Args) > 1 {
		input = os.Args[1]
	}
	fmt.Printf("User: %s\n", input)

	response, err := myAgent.Run(ctx, input, nil)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	fmt.Printf("Agent: %s\n", response)
}
