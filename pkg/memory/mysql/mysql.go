package mysql

import (
	"fmt"

	gormmem "github.com/barekit/talos/pkg/memory/gorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// New creates a new MySQL memory.
func New(dsn string) (*gormmem.Memory, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %w", err)
	}
	return gormmem.New(db)
}
