package postgres

import (
	"fmt"

	gormmem "github.com/barekit/talos/pkg/memory/gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// New creates a new Postgres memory.
func New(dsn string) (*gormmem.Memory, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}
	return gormmem.New(db)
}
