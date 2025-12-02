package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Embedder implements knowledge.Embedder using OpenAI.
type Embedder struct {
	client *openai.Client
	model  openai.EmbeddingModel
}

// NewEmbedder creates a new OpenAI Embedder.
func NewEmbedder(opts ...option.RequestOption) *Embedder {
	client := openai.NewClient(opts...)
	return &Embedder{
		client: &client,
		model:  openai.EmbeddingModelTextEmbedding3Small,
	}
}

// Embed generates embeddings for the given texts.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	params := openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: e.model,
	}

	resp, err := e.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		// Convert []float64 to []float32
		vec := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vec[j] = float32(v)
		}
		embeddings[i] = vec
	}

	return embeddings, nil
}
