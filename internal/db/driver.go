package db

import (
	"context"
	"database-mcp/internal/types"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeClickHouse DatabaseType = "clickhouse"
)

// DatabaseDriver interface defines the contract for database drivers
type DatabaseDriver interface {
	// Connection management
	Connect(ctx context.Context, dsn string) error
	Close() error
	Ping(ctx context.Context) error

	// Schema operations
	ListSchemas(ctx context.Context) ([]types.Schema, error)
	ListTables(ctx context.Context, schema string) ([]types.Table, error)
	DescribeTable(ctx context.Context, schema, table string) (*types.TableDescription, error)

	// Query operations
	RunSQL(ctx context.Context, query string, limit int) (*types.QueryResult, error)
	ExplainQuery(ctx context.Context, query string) (*types.ExplainResult, error)

	// Metadata
	GetType() DatabaseType
	GetVersion(ctx context.Context) (string, error)
}
