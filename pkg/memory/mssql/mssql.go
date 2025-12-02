package mssql

import (
	"fmt"

	gormmem "github.com/barekit/talos/pkg/memory/gorm"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// New creates a new MSSQL memory.
func New(dsn string) (*gormmem.Memory, error) {
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open mssql: %w", err)
	}
	return gormmem.New(db)
}
