package sqlite

import (
	"fmt"

	gormmem "github.com/barekit/talos/pkg/memory/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// New creates a new SQLite memory.
func New(dsn string) (*gormmem.Memory, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}
	return gormmem.New(db)
}
