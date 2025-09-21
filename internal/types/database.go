package types

// Database-specific types

// Schema represents a database schema
type Schema struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Table represents a database table
type Table struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Type        string `json:"type"` // "table", "view", "materialized_view"
	Description string `json:"description,omitempty"`
	RowCount    *int64 `json:"rowCount,omitempty"`
}

// Column represents a database column
type Column struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
	IsPrimaryKey bool   `json:"isPrimaryKey,omitempty"`
	IsForeignKey bool   `json:"isForeignKey,omitempty"`
	IsIndex      bool   `json:"isIndex,omitempty"`
	IsGeometry   bool   `json:"isGeometry,omitempty"` // For PostGIS columns
}

// TableDescription represents detailed table information
type TableDescription struct {
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Columns     []Column `json:"columns"`
	Indexes     []Index  `json:"indexes,omitempty"`
	RowCount    *int64   `json:"rowCount,omitempty"`
}

// Index represents a database index
type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"isUnique"`
	Type     string   `json:"type,omitempty"`
}

// QueryResult represents the result of running a SQL query
type QueryResult struct {
	Columns         []string        `json:"columns"`
	Rows            [][]interface{} `json:"rows"`
	RowCount        int             `json:"rowCount"`
	ExecutionTimeMs int64           `json:"executionTimeMs"`
	Query           string          `json:"query"`
}

// ExplainResult represents the result of explaining a SQL query
type ExplainResult struct {
	Query           string                 `json:"query"`
	Plan            map[string]interface{} `json:"plan"`
	ExecutionTimeMs int64                  `json:"executionTimeMs"`
}
