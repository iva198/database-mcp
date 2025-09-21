package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"database-mcp/internal/types"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseDriver implements DatabaseDriver for ClickHouse
type ClickHouseDriver struct {
	conn clickhouse.Conn
	dsn  string
}

// NewClickHouseDriver creates a new ClickHouse driver
func NewClickHouseDriver() DatabaseDriver {
	return &ClickHouseDriver{}
}

// Connect establishes a connection to ClickHouse
func (d *ClickHouseDriver) Connect(ctx context.Context, dsn string) error {
	d.dsn = dsn

	// Parse ClickHouse options from DSN
	options, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse ClickHouse DSN: %w", err)
	}

	// Set connection pool settings
	options.MaxOpenConns = 10
	options.MaxIdleConns = 5
	options.ConnMaxLifetime = time.Hour

	// Create connection
	conn, err := clickhouse.Open(options)
	if err != nil {
		return fmt.Errorf("failed to create ClickHouse connection: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping ClickHouse database: %w", err)
	}

	d.conn = conn
	slog.Info("Connected to ClickHouse", "dsn", maskClickHouseDSN(dsn))
	return nil
}

// Close closes the ClickHouse connection
func (d *ClickHouseDriver) Close() error {
	if d.conn != nil {
		err := d.conn.Close()
		d.conn = nil
		slog.Info("Closed ClickHouse connection")
		return err
	}
	return nil
}

// Ping checks if the ClickHouse connection is alive
func (d *ClickHouseDriver) Ping(ctx context.Context) error {
	if d.conn == nil {
		return fmt.Errorf("ClickHouse not connected")
	}
	return d.conn.Ping(ctx)
}

// ListSchemas lists all ClickHouse databases (schemas)
func (d *ClickHouseDriver) ListSchemas(ctx context.Context) ([]types.Schema, error) {
	if d.conn == nil {
		return nil, fmt.Errorf("ClickHouse not connected")
	}

	query := `
		SELECT 
			name,
			comment
		FROM system.databases 
		WHERE name NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA')
		ORDER BY name`

	rows, err := d.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []types.Schema
	for rows.Next() {
		var schema types.Schema
		if err := rows.Scan(&schema.Name, &schema.Description); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	return schemas, nil
}

// ListTables lists tables in a ClickHouse database
func (d *ClickHouseDriver) ListTables(ctx context.Context, schema string) ([]types.Table, error) {
	if d.conn == nil {
		return nil, fmt.Errorf("ClickHouse not connected")
	}

	query := `
		SELECT 
			name,
			database,
			CASE 
				WHEN engine LIKE '%View' THEN 'view'
				WHEN engine = 'MaterializedView' THEN 'materialized_view'
				ELSE 'table'
			END as table_type,
			comment,
			total_rows
		FROM system.tables 
		WHERE database = ?
		ORDER BY name`

	rows, err := d.conn.Query(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []types.Table
	for rows.Next() {
		var table types.Table
		var totalRows uint64
		if err := rows.Scan(&table.Name, &table.Schema, &table.Type, &table.Description, &totalRows); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		// Convert uint64 to *int64
		if totalRows > 0 {
			rowCount := int64(totalRows)
			table.RowCount = &rowCount
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// DescribeTable describes a ClickHouse table
func (d *ClickHouseDriver) DescribeTable(ctx context.Context, schema, table string) (*types.TableDescription, error) {
	if d.conn == nil {
		return nil, fmt.Errorf("ClickHouse not connected")
	}

	// Get table info
	tableQuery := `
		SELECT 
			database,
			name,
			CASE 
				WHEN engine LIKE '%View' THEN 'view'
				WHEN engine = 'MaterializedView' THEN 'materialized_view'
				ELSE 'table'
			END as table_type,
			comment,
			total_rows
		FROM system.tables 
		WHERE database = ? AND name = ?`

	var desc types.TableDescription
	var totalRows uint64
	err := d.conn.QueryRow(ctx, tableQuery, schema, table).Scan(
		&desc.Schema, &desc.Name, &desc.Type, &desc.Description, &totalRows,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query table info: %w", err)
	}

	// Convert uint64 to *int64
	if totalRows > 0 {
		rowCount := int64(totalRows)
		desc.RowCount = &rowCount
	}

	// Get columns
	columnQuery := `
		SELECT 
			name,
			type,
			is_in_partition_key,
			is_in_sorting_key,
			is_in_primary_key,
			is_in_sampling_key,
			default_expression,
			comment
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position`

	rows, err := d.conn.Query(ctx, columnQuery, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col types.Column
		var isInPartitionKey, isInSortingKey, isInPrimaryKey, isInSamplingKey uint8
		var defaultExpr, comment string

		if err := rows.Scan(
			&col.Name, &col.Type, &isInPartitionKey, &isInSortingKey,
			&isInPrimaryKey, &isInSamplingKey, &defaultExpr, &comment,
		); err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}

		// ClickHouse doesn't have nullable concept like PostgreSQL
		col.Nullable = strings.Contains(strings.ToLower(col.Type), "nullable")
		col.DefaultValue = defaultExpr
		col.Description = comment
		col.IsPrimaryKey = isInPrimaryKey > 0
		col.IsIndex = isInSortingKey > 0 || isInPrimaryKey > 0

		desc.Columns = append(desc.Columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	// ClickHouse doesn't have traditional indexes like PostgreSQL
	// But we can show sorting keys and primary keys as "indexes"
	if len(desc.Columns) > 0 {
		var primaryKeyCols []string
		var sortingKeyCols []string

		for _, col := range desc.Columns {
			if col.IsPrimaryKey {
				primaryKeyCols = append(primaryKeyCols, col.Name)
			}
			if col.IsIndex && !col.IsPrimaryKey {
				sortingKeyCols = append(sortingKeyCols, col.Name)
			}
		}

		if len(primaryKeyCols) > 0 {
			desc.Indexes = append(desc.Indexes, types.Index{
				Name:     "PRIMARY",
				Columns:  primaryKeyCols,
				IsUnique: true,
				Type:     "primary_key",
			})
		}

		if len(sortingKeyCols) > 0 {
			desc.Indexes = append(desc.Indexes, types.Index{
				Name:     "SORTING_KEY",
				Columns:  sortingKeyCols,
				IsUnique: false,
				Type:     "sorting_key",
			})
		}
	}

	return &desc, nil
}

// RunSQL executes a SQL query on ClickHouse
func (d *ClickHouseDriver) RunSQL(ctx context.Context, query string, limit int) (*types.QueryResult, error) {
	if d.conn == nil {
		return nil, fmt.Errorf("ClickHouse not connected")
	}

	startTime := time.Now()

	// Add LIMIT if not present (basic implementation)
	if limit > 0 && !strings.Contains(strings.ToUpper(query), " LIMIT ") {
		query = fmt.Sprintf("%s LIMIT %d", strings.TrimRight(query, ";"), limit)
	}

	rows, err := d.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columnTypes := rows.ColumnTypes()
	columns := make([]string, len(columnTypes))
	for i, ct := range columnTypes {
		columns[i] = ct.Name()
	}

	// Read all rows
	var resultRows [][]interface{}
	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row values: %w", err)
		}

		resultRows = append(resultRows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	executionTime := time.Since(startTime)

	return &types.QueryResult{
		Columns:         columns,
		Rows:            resultRows,
		RowCount:        len(resultRows),
		ExecutionTimeMs: executionTime.Milliseconds(),
		Query:           query,
	}, nil
}

// ExplainQuery explains a SQL query on ClickHouse
func (d *ClickHouseDriver) ExplainQuery(ctx context.Context, query string) (*types.ExplainResult, error) {
	if d.conn == nil {
		return nil, fmt.Errorf("ClickHouse not connected")
	}

	startTime := time.Now()

	// ClickHouse supports EXPLAIN AST and EXPLAIN PLAN
	explainQuery := fmt.Sprintf("EXPLAIN AST %s", query)

	var explainResult string
	err := d.conn.QueryRow(ctx, explainQuery).Scan(&explainResult)
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}

	executionTime := time.Since(startTime)

	// Also try to get the execution plan
	planQuery := fmt.Sprintf("EXPLAIN PLAN %s", query)
	planRows, err := d.conn.Query(ctx, planQuery)
	var planLines []string
	if err == nil {
		defer planRows.Close()
		for planRows.Next() {
			var line string
			if planRows.Scan(&line) == nil {
				planLines = append(planLines, line)
			}
		}
	}

	plan := map[string]interface{}{
		"format": "clickhouse_ast",
		"ast":    explainResult,
	}

	if len(planLines) > 0 {
		plan["execution_plan"] = planLines
	}

	return &types.ExplainResult{
		Query:           query,
		Plan:            plan,
		ExecutionTimeMs: executionTime.Milliseconds(),
	}, nil
}

// GetType returns the database type
func (d *ClickHouseDriver) GetType() DatabaseType {
	return DatabaseTypeClickHouse
}

// GetVersion returns the ClickHouse version
func (d *ClickHouseDriver) GetVersion(ctx context.Context) (string, error) {
	if d.conn == nil {
		return "ClickHouse (not connected)", nil
	}

	var version string
	err := d.conn.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return "ClickHouse (version unknown)", nil
	}

	return fmt.Sprintf("ClickHouse %s", version), nil
}

// maskClickHouseDSN masks sensitive information in DSN for logging
func maskClickHouseDSN(dsn string) string {
	// Simple masking for ClickHouse DSN
	if strings.Contains(dsn, "password=") {
		parts := strings.Split(dsn, "&")
		for i, part := range parts {
			if strings.HasPrefix(part, "password=") {
				parts[i] = "password=***"
			}
		}
		return strings.Join(parts, "&")
	}
	return dsn
}
