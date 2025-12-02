package knowledge

import (
	"context"
)

// Document represents a piece of text with metadata.
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float32                `json:"score,omitempty"` // Similarity score
}

// Embedder is the interface for generating embeddings.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// VectorStore is the interface for storing and retrieving vectors.
type VectorStore interface {
	// Upsert inserts or updates documents and their vectors.
	Upsert(ctx context.Context, vectors [][]float32, documents []Document) error
	// Search searches for similar documents using a query vector.
	Search(ctx context.Context, query []float32, limit int) ([]Document, error)
}

// KnowledgeBase combines an Embedder and a VectorStore.
type KnowledgeBase struct {
	Embedder    Embedder
	VectorStore VectorStore
}

// NewKnowledgeBase creates a new KnowledgeBase.
func NewKnowledgeBase(embedder Embedder, store VectorStore) *KnowledgeBase {
	return &KnowledgeBase{
		Embedder:    embedder,
		VectorStore: store,
	}
}

// Ingest adds texts to the knowledge base.
func (kb *KnowledgeBase) Ingest(ctx context.Context, docs []Document) error {
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	vectors, err := kb.Embedder.Embed(ctx, texts)
	if err != nil {
		return err
	}

	return kb.VectorStore.Upsert(ctx, vectors, docs)
}

// Retrieve finds relevant documents for a query.
func (kb *KnowledgeBase) Retrieve(ctx context.Context, query string, limit int) ([]Document, error) {
	vectors, err := kb.Embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, nil
	}

	return kb.VectorStore.Search(ctx, vectors[0], limit)
}
