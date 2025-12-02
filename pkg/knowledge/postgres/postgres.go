package postgres

import (
	"context"
	"fmt"

	"github.com/barekit/talos/pkg/knowledge"
	"github.com/pgvector/pgvector-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresStore implements knowledge.VectorStore using pgvector.
type PostgresStore struct {
	db *gorm.DB
}

// DocumentModel represents the database schema for a document.
type DocumentModel struct {
	ID        string `gorm:"primaryKey"`
	Content   string
	Metadata  []byte          `gorm:"type:jsonb"`        // Store metadata as JSONB
	Embedding pgvector.Vector `gorm:"type:vector(1536)"` // Adjust dimension as needed
}

// TableName overrides the table name.
func (DocumentModel) TableName() string {
	return "documents"
}

// New creates a new PostgresStore.
func New(dsn string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return nil, fmt.Errorf("failed to enable pgvector extension: %w", err)
	}

	// AutoMigrate
	if err := db.AutoMigrate(&DocumentModel{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Upsert(ctx context.Context, vectors [][]float32, documents []knowledge.Document) error {
	if len(vectors) != len(documents) {
		return fmt.Errorf("number of vectors and documents must match")
	}

	// Use a transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, doc := range documents {
			// Convert metadata to JSON bytes (simplified)
			// In a real app, use json.Marshal
			metadataJSON := []byte("{}")

			model := DocumentModel{
				ID:        doc.ID,
				Content:   doc.Content,
				Metadata:  metadataJSON,
				Embedding: pgvector.NewVector(vectors[i]),
			}

			// Upsert
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"content", "metadata", "embedding"}),
			}).Create(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PostgresStore) Search(ctx context.Context, query []float32, limit int) ([]knowledge.Document, error) {
	var models []DocumentModel

	// Cosine distance: 1 - (A . B) / (|A| * |B|)
	// pgvector operator for cosine distance is <=>
	// We order by distance ascending

	err := s.db.WithContext(ctx).
		Order(clause.Expr{SQL: "embedding <=> ?", Vars: []interface{}{pgvector.NewVector(query)}}).
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	docs := make([]knowledge.Document, len(models))
	for i, m := range models {
		docs[i] = knowledge.Document{
			ID:      m.ID,
			Content: m.Content,
			// Metadata: unmarshal m.Metadata
		}
	}

	return docs, nil
}
