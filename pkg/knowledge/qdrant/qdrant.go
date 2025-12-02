package qdrant

import (
	"context"
	"fmt"

	"github.com/barekit/talos/pkg/knowledge"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantStore implements knowledge.VectorStore using Qdrant.
type QdrantStore struct {
	client         *qdrant.Client
	collectionName string
	vectorSize     uint64
}

// New creates a new QdrantStore.
func New(host string, port int, collectionName string, vectorSize uint64) (*QdrantStore, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	store := &QdrantStore{
		client:         client,
		collectionName: collectionName,
		vectorSize:     vectorSize,
	}

	if err := store.initCollection(context.Background()); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *QdrantStore) initCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := s.client.CollectionExists(ctx, s.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		// Create collection
		err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: s.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     s.vectorSize,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

func (s *QdrantStore) Upsert(ctx context.Context, vectors [][]float32, documents []knowledge.Document) error {
	if len(vectors) != len(documents) {
		return fmt.Errorf("number of vectors and documents must match")
	}

	points := make([]*qdrant.PointStruct, len(vectors))
	for i, doc := range documents {
		// Convert metadata to map[string]*qdrant.Value
		payload := make(map[string]*qdrant.Value)
		payload["content"] = qdrant.NewValueString(doc.Content)
		for k, v := range doc.Metadata {
			// Simple conversion for strings, can be expanded
			if strVal, ok := v.(string); ok {
				payload[k] = qdrant.NewValueString(strVal)
			}
		}

		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(doc.ID),
			Vectors: qdrant.NewVectors(vectors[i]...),
			Payload: payload,
		}
	}

	wait := true
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points:         points,
		Wait:           &wait,
	})
	return err
}

func (s *QdrantStore) Search(ctx context.Context, query []float32, limit int) ([]knowledge.Document, error) {
	limit64 := uint64(limit)
	res, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName,
		Query:          qdrant.NewQuery(query...),
		Limit:          &limit64,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	docs := make([]knowledge.Document, len(res))
	for i, hit := range res {
		content := ""
		if c, ok := hit.Payload["content"]; ok {
			content = c.GetStringValue()
		}

		metadata := make(map[string]interface{})
		for k, v := range hit.Payload {
			if k != "content" {
				metadata[k] = v.GetStringValue() // Simplified
			}
		}

		docs[i] = knowledge.Document{
			ID:       hit.Id.GetUuid(),
			Content:  content,
			Metadata: metadata,
			Score:    hit.Score,
		}
	}

	return docs, nil
}
