package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"database-mcp/internal/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQLDriver implements DatabaseDriver for PostgreSQL
type PostgreSQLDriver struct {
	pool *pgxpool.Pool
	dsn  string
}

// NewPostgreSQLDriver creates a new PostgreSQL driver
func NewPostgreSQLDriver() DatabaseDriver {
	return &PostgreSQLDriver{}
}

// Connect establishes a connection to PostgreSQL
func (d *PostgreSQLDriver) Connect(ctx context.Context, dsn string) error {
	d.dsn = dsn

	// Parse and create connection pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse PostgreSQL DSN: %w", err)
	}

	// Configure connection pool
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	d.pool = pool
	slog.Info("Connected to PostgreSQL", "dsn", maskDSN(dsn))
	return nil
}

// Close closes the PostgreSQL connection
func (d *PostgreSQLDriver) Close() error {
	if d.pool != nil {
		d.pool.Close()
		d.pool = nil
		slog.Info("Closed PostgreSQL connection")
	}
	return nil
}

// Ping checks if the PostgreSQL connection is alive
func (d *PostgreSQLDriver) Ping(ctx context.Context) error {
	if d.pool == nil {
		return fmt.Errorf("PostgreSQL not connected")
	}
	return d.pool.Ping(ctx)
}

// ListSchemas lists all PostgreSQL schemas
func (d *PostgreSQLDriver) ListSchemas(ctx context.Context) ([]types.Schema, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("PostgreSQL not connected")
	}

	query := `
		SELECT 
			schema_name,
			COALESCE(pg_catalog.obj_description(n.oid, 'pg_namespace'), '') as description
		FROM information_schema.schemata s
		LEFT JOIN pg_catalog.pg_namespace n ON n.nspname = s.schema_name
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND schema_name NOT LIKE 'pg_temp_%'
		  AND schema_name NOT LIKE 'pg_toast_temp_%'
		ORDER BY schema_name`

	rows, err := d.pool.Query(ctx, query)
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

// ListTables lists tables in a PostgreSQL schema
func (d *PostgreSQLDriver) ListTables(ctx context.Context, schema string) ([]types.Table, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("PostgreSQL not connected")
	}

	query := `
		SELECT 
			t.table_name,
			t.table_schema,
			CASE 
				WHEN t.table_type = 'BASE TABLE' THEN 'table'
				WHEN t.table_type = 'VIEW' THEN 'view'
				ELSE LOWER(t.table_type)
			END as table_type,
			COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '') as description,
			CASE 
				WHEN s.n_tup_ins IS NOT NULL THEN s.n_tup_ins + s.n_tup_upd + s.n_tup_del
				ELSE NULL
			END as row_count
		FROM information_schema.tables t
		LEFT JOIN pg_catalog.pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		LEFT JOIN pg_catalog.pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema = $1
		ORDER BY t.table_name`

	rows, err := d.pool.Query(ctx, query, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []types.Table
	for rows.Next() {
		var table types.Table
		var rowCount *int64
		if err := rows.Scan(&table.Name, &table.Schema, &table.Type, &table.Description, &rowCount); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}
		table.RowCount = rowCount
		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// DescribeTable describes a PostgreSQL table
func (d *PostgreSQLDriver) DescribeTable(ctx context.Context, schema, table string) (*types.TableDescription, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("PostgreSQL not connected")
	}

	// Get table info
	tableQuery := `
		SELECT 
			t.table_schema,
			t.table_name,
			CASE 
				WHEN t.table_type = 'BASE TABLE' THEN 'table'
				WHEN t.table_type = 'VIEW' THEN 'view'
				ELSE LOWER(t.table_type)
			END as table_type,
			COALESCE(pg_catalog.obj_description(c.oid, 'pg_class'), '') as description,
			CASE 
				WHEN s.n_tup_ins IS NOT NULL THEN s.n_tup_ins + s.n_tup_upd + s.n_tup_del
				ELSE NULL
			END as row_count
		FROM information_schema.tables t
		LEFT JOIN pg_catalog.pg_class c ON c.relname = t.table_name
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace AND n.nspname = t.table_schema
		LEFT JOIN pg_catalog.pg_stat_user_tables s ON s.relname = t.table_name AND s.schemaname = t.table_schema
		WHERE t.table_schema = $1 AND t.table_name = $2`

	var desc types.TableDescription
	var rowCount *int64
	err := d.pool.QueryRow(ctx, tableQuery, schema, table).Scan(
		&desc.Schema, &desc.Name, &desc.Type, &desc.Description, &rowCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query table info: %w", err)
	}
	desc.RowCount = rowCount

	// Get columns
	columnQuery := `
		SELECT 
			c.column_name,
			c.data_type,
			CASE WHEN c.is_nullable = 'YES' THEN true ELSE false END as is_nullable,
			COALESCE(c.column_default, '') as column_default,
			COALESCE(pgd.description, '') as description,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary_key,
			CASE WHEN fk.column_name IS NOT NULL THEN true ELSE false END as is_foreign_key,
			CASE WHEN idx.column_name IS NOT NULL THEN true ELSE false END as is_indexed,
			CASE WHEN geo.column_name IS NOT NULL THEN true ELSE false END as is_geometry
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_class pgc ON pgc.relname = c.table_name
		LEFT JOIN pg_catalog.pg_namespace pgn ON pgn.oid = pgc.relnamespace AND pgn.nspname = c.table_schema
		LEFT JOIN pg_catalog.pg_attribute pga ON pga.attrelid = pgc.oid AND pga.attname = c.column_name
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = pgc.oid AND pgd.objsubid = pga.attnum
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON pk.column_name = c.column_name
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'FOREIGN KEY'
		) fk ON fk.column_name = c.column_name
		LEFT JOIN (
			SELECT DISTINCT a.attname as column_name
			FROM pg_catalog.pg_index i
			JOIN pg_catalog.pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			JOIN pg_catalog.pg_class t ON t.oid = i.indrelid
			JOIN pg_catalog.pg_namespace n ON n.oid = t.relnamespace
			WHERE n.nspname = $1 AND t.relname = $2
		) idx ON idx.column_name = c.column_name
		LEFT JOIN (
			SELECT c.column_name
			FROM information_schema.columns c
			WHERE c.table_schema = $1 AND c.table_name = $2 
			  AND c.udt_name = 'geometry'
		) geo ON geo.column_name = c.column_name
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := d.pool.Query(ctx, columnQuery, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col types.Column
		if err := rows.Scan(
			&col.Name, &col.Type, &col.Nullable, &col.DefaultValue, &col.Description,
			&col.IsPrimaryKey, &col.IsForeignKey, &col.IsIndex, &col.IsGeometry,
		); err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}
		desc.Columns = append(desc.Columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	// Get indexes
	indexQuery := `
		SELECT 
			i.relname as index_name,
			array_agg(a.attname ORDER BY a.attnum) as column_names,
			ix.indisunique as is_unique,
			CASE 
				WHEN ix.indisunique THEN 'unique'
				WHEN am.amname = 'gist' THEN 'gist'
				WHEN am.amname = 'gin' THEN 'gin'
				ELSE 'btree'
			END as index_type
		FROM pg_catalog.pg_index ix
		JOIN pg_catalog.pg_class i ON i.oid = ix.indexrelid
		JOIN pg_catalog.pg_class t ON t.oid = ix.indrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_catalog.pg_am am ON am.oid = i.relam
		WHERE n.nspname = $1 AND t.relname = $2 AND NOT ix.indisprimary
		GROUP BY i.relname, ix.indisunique, am.amname
		ORDER BY i.relname`

	indexRows, err := d.pool.Query(ctx, indexQuery, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var idx types.Index
		var columnNames []string
		if err := indexRows.Scan(&idx.Name, &columnNames, &idx.IsUnique, &idx.Type); err != nil {
			return nil, fmt.Errorf("failed to scan index row: %w", err)
		}
		idx.Columns = columnNames
		desc.Indexes = append(desc.Indexes, idx)
	}

	return &desc, nil
}

// RunSQL executes a SQL query on PostgreSQL
func (d *PostgreSQLDriver) RunSQL(ctx context.Context, query string, limit int) (*types.QueryResult, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("PostgreSQL not connected")
	}

	startTime := time.Now()

	// Add LIMIT if not present (basic implementation)
	if limit > 0 && !strings.Contains(strings.ToUpper(query), " LIMIT ") {
		query = fmt.Sprintf("%s LIMIT %d", strings.TrimRight(query, ";"), limit)
	}

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Read all rows
	var resultRows [][]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
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

// ExplainQuery explains a SQL query on PostgreSQL
func (d *PostgreSQLDriver) ExplainQuery(ctx context.Context, query string) (*types.ExplainResult, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("PostgreSQL not connected")
	}

	startTime := time.Now()
	explainQuery := fmt.Sprintf("EXPLAIN (FORMAT JSON, ANALYZE FALSE, VERBOSE TRUE, BUFFERS FALSE) %s", query)

	var planJSON string
	err := d.pool.QueryRow(ctx, explainQuery).Scan(&planJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}

	executionTime := time.Since(startTime)

	// Parse the JSON plan (simplified - just store as map)
	plan := map[string]interface{}{
		"format": "postgresql_json",
		"raw":    planJSON,
	}

	return &types.ExplainResult{
		Query:           query,
		Plan:            plan,
		ExecutionTimeMs: executionTime.Milliseconds(),
	}, nil
}

// GetType returns the database type
func (d *PostgreSQLDriver) GetType() DatabaseType {
	return DatabaseTypePostgreSQL
}

// GetVersion returns the PostgreSQL version
func (d *PostgreSQLDriver) GetVersion(ctx context.Context) (string, error) {
	if d.pool == nil {
		return "PostgreSQL (not connected)", nil
	}

	var version string
	err := d.pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		return "PostgreSQL (version unknown)", nil
	}

	return version, nil
}

// maskDSN masks sensitive information in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking - replace password with ***
	if strings.Contains(dsn, ":") && strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		if len(parts) >= 2 {
			userPart := parts[0]
			if strings.Contains(userPart, ":") {
				userParts := strings.Split(userPart, ":")
				if len(userParts) >= 3 {
					userParts[len(userParts)-1] = "***"
					parts[0] = strings.Join(userParts, ":")
					return strings.Join(parts, "@")
				}
			}
		}
	}
	return dsn
}
