package memory

import (
	"context"
	"fmt"

	"github.com/barekit/talos/pkg/memory/consts"
	"github.com/barekit/talos/pkg/memory/inmemory"
	mongomem "github.com/barekit/talos/pkg/memory/mongo"
	"github.com/barekit/talos/pkg/memory/mssql"
	"github.com/barekit/talos/pkg/memory/mysql"
	"github.com/barekit/talos/pkg/memory/neo4j"
	"github.com/barekit/talos/pkg/memory/postgres"
	"github.com/barekit/talos/pkg/memory/redis"
	"github.com/barekit/talos/pkg/memory/sqlite"
	goredis "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Type string

const (
	TypeSQLite   Type = "sqlite"
	TypePostgres Type = "postgres"
	TypeMySQL    Type = "mysql"
	TypeMSSQL    Type = "mssql"
	TypeRedis    Type = "redis"
	TypeNeo4j    Type = "neo4j"
	TypeMongo    Type = "mongo"
	TypeInMemory Type = "inmemory"
)

// Config holds configuration for memory adapters.
type Config struct {
	Type             Type
	ConnectionString string
	Username         string
	Password         string
	DBName           string
	// Additional options can be added here (e.g., Redis options, Neo4j auth)
}

// NewFactory creates a new memory adapter based on the configuration.
func NewFactory(ctx context.Context, cfg Config) (Memory, error) {
	switch cfg.Type {
	case TypeSQLite:
		return sqlite.New(cfg.ConnectionString)

	case TypePostgres:
		return postgres.New(cfg.ConnectionString)

	case TypeRedis:
		opts, err := goredis.ParseURL(cfg.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis url: %w", err)
		}
		client := goredis.NewClient(opts)
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to ping redis: %w", err)
		}
		return redis.New(client), nil

	case TypeNeo4j:
		dbName := "neo4j" // Default Neo4j DB is typically "neo4j", not "talos"
		if cfg.DBName != "" {
			dbName = cfg.DBName
		}
		return neo4j.New(cfg.ConnectionString, cfg.Username, cfg.Password, dbName)

	case TypeMongo:
		opts := options.Client().ApplyURI(cfg.ConnectionString)
		client, err := mongo.Connect(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to mongo: %w", err)
		}
		if err := client.Ping(ctx, nil); err != nil {
			return nil, fmt.Errorf("failed to ping mongo: %w", err)
		}
		dbName := consts.DefaultDBName
		if cfg.DBName != "" {
			dbName = cfg.DBName
		}
		return mongomem.New(client, dbName, consts.TableNameMessages), nil

	case TypeMySQL:
		return mysql.New(cfg.ConnectionString)

	case TypeMSSQL:
		return mssql.New(cfg.ConnectionString)

	case TypeInMemory:
		return inmemory.New(), nil

	default:
		return nil, fmt.Errorf("unsupported memory type: %s", cfg.Type)
	}
}
