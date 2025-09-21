package db

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"database-mcp/internal/types"
)

// Manager manages multiple database connections
type Manager struct {
	primaryDriver   DatabaseDriver
	analyticsDriver DatabaseDriver
	primaryURL      string
	analyticsURL    string
}

// NewManager creates a new database manager
func NewManager(primaryURL, analyticsURL string) (*Manager, error) {
	if primaryURL == "" {
		return nil, fmt.Errorf("primary database URL is required")
	}

	return &Manager{
		primaryURL:   primaryURL,
		analyticsURL: analyticsURL,
	}, nil
}

// Connect establishes connections to all configured databases
func (m *Manager) Connect(ctx context.Context) error {
	// Connect to primary database
	primaryDriver, err := createDriver(m.primaryURL)
	if err != nil {
		return fmt.Errorf("failed to create primary driver: %w", err)
	}

	if err := primaryDriver.Connect(ctx, m.primaryURL); err != nil {
		return fmt.Errorf("failed to connect to primary database: %w", err)
	}

	m.primaryDriver = primaryDriver
	slog.Info("Connected to primary database", "type", primaryDriver.GetType())

	// Connect to analytics database if configured
	if m.analyticsURL != "" {
		analyticsDriver, err := createDriver(m.analyticsURL)
		if err != nil {
			slog.Warn("Failed to create analytics driver", "error", err)
		} else {
			if err := analyticsDriver.Connect(ctx, m.analyticsURL); err != nil {
				slog.Warn("Failed to connect to analytics database", "error", err)
			} else {
				m.analyticsDriver = analyticsDriver
				slog.Info("Connected to analytics database", "type", analyticsDriver.GetType())
			}
		}
	}

	return nil
}

// Close closes all database connections
func (m *Manager) Close() error {
	var errs []error

	if m.primaryDriver != nil {
		if err := m.primaryDriver.Close(); err != nil {
			errs = append(errs, fmt.Errorf("primary database close error: %w", err))
		}
	}

	if m.analyticsDriver != nil {
		if err := m.analyticsDriver.Close(); err != nil {
			errs = append(errs, fmt.Errorf("analytics database close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("database close errors: %v", errs)
	}

	return nil
}

// GetDriver returns the appropriate driver based on database name
func (m *Manager) GetDriver(database string) (DatabaseDriver, error) {
	switch database {
	case "primary", "":
		if m.primaryDriver == nil {
			return nil, fmt.Errorf("primary database not connected")
		}
		return m.primaryDriver, nil
	case "analytics":
		if m.analyticsDriver == nil {
			return nil, fmt.Errorf("analytics database not connected or configured")
		}
		return m.analyticsDriver, nil
	default:
		return nil, fmt.Errorf("unknown database: %s", database)
	}
}

// ListSchemas lists schemas from the specified database
func (m *Manager) ListSchemas(ctx context.Context, database string) ([]types.Schema, error) {
	driver, err := m.GetDriver(database)
	if err != nil {
		return nil, err
	}
	return driver.ListSchemas(ctx)
}

// ListTables lists tables from the specified database and schema
func (m *Manager) ListTables(ctx context.Context, database, schema string) ([]types.Table, error) {
	driver, err := m.GetDriver(database)
	if err != nil {
		return nil, err
	}
	return driver.ListTables(ctx, schema)
}

// DescribeTable describes a table from the specified database
func (m *Manager) DescribeTable(ctx context.Context, database, schema, table string) (*types.TableDescription, error) {
	driver, err := m.GetDriver(database)
	if err != nil {
		return nil, err
	}
	return driver.DescribeTable(ctx, schema, table)
}

// RunSQL executes a SQL query on the specified database
func (m *Manager) RunSQL(ctx context.Context, database, query string, limit int) (*types.QueryResult, error) {
	driver, err := m.GetDriver(database)
	if err != nil {
		return nil, err
	}
	return driver.RunSQL(ctx, query, limit)
}

// ExplainQuery explains a SQL query on the specified database
func (m *Manager) ExplainQuery(ctx context.Context, database, query string) (*types.ExplainResult, error) {
	driver, err := m.GetDriver(database)
	if err != nil {
		return nil, err
	}
	return driver.ExplainQuery(ctx, query)
}

// GetDatabaseInfo returns information about connected databases
func (m *Manager) GetDatabaseInfo(ctx context.Context) map[string]interface{} {
	info := make(map[string]interface{})

	if m.primaryDriver != nil {
		version, _ := m.primaryDriver.GetVersion(ctx)
		info["primary"] = map[string]interface{}{
			"type":    m.primaryDriver.GetType(),
			"version": version,
		}
	}

	if m.analyticsDriver != nil {
		version, _ := m.analyticsDriver.GetVersion(ctx)
		info["analytics"] = map[string]interface{}{
			"type":    m.analyticsDriver.GetType(),
			"version": version,
		}
	}

	return info
}

// createDriver creates a database driver based on the connection URL
func createDriver(dsn string) (DatabaseDriver, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "postgres", "postgresql":
		return NewPostgreSQLDriver(), nil
	case "clickhouse":
		return NewClickHouseDriver(), nil
	default:
		return nil, fmt.Errorf("unsupported database scheme: %s", scheme)
	}
}
