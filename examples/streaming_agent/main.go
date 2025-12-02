package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/barekit/talos/pkg/agent"
	"github.com/barekit/talos/pkg/llm"
	"github.com/barekit/talos/pkg/llm/openai"
)

func main() {
	ctx := context.Background()

	// Initialize LLM
	llmProvider := openai.New()

	// Initialize Agent
	myAgent := agent.New(
		llmProvider,
		agent.WithInstructions("You are a helpful assistant. If the user provides an image, describe it."),
	)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nUser (type 'exit' to quit): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}

		var attachments []llm.Attachment

		// Check for image command: /image <url> <prompt>
		if strings.HasPrefix(input, "/image ") {
			parts := strings.SplitN(input, " ", 3)
			if len(parts) >= 2 {
				url := parts[1]
				prompt := "Describe this image."
				if len(parts) > 2 {
					prompt = parts[2]
				}

				attachments = append(attachments, llm.Attachment{
					Type: "image_url",
					URL:  url,
				})
				input = prompt
				fmt.Printf("Attached image: %s\n", url)
			}
		}

		fmt.Print("Assistant: ")
		stream, err := myAgent.RunStream(ctx, input, attachments)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		for chunk := range stream {
			fmt.Print(chunk)
		}
		fmt.Println()
	}
}
