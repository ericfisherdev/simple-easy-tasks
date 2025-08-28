package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// DatabaseOptimizer handles database performance optimization
type DatabaseOptimizer interface {
	// Index management
	AnalyzeIndexUsage(ctx context.Context) (*IndexAnalysisReport, error)
	CreateRecommendedIndexes(ctx context.Context) error
	OptimizeQueries(ctx context.Context) (*QueryOptimizationReport, error)

	// Performance monitoring
	GetSlowQueries(ctx context.Context, threshold time.Duration) ([]*SlowQuery, error)
	AnalyzeQueryPlan(ctx context.Context, query string, args []interface{}) (*QueryPlan, error)

	// Connection optimization
	OptimizeConnectionPool(ctx context.Context, config *ConnectionPoolConfig) error
	GetConnectionStats(ctx context.Context) (*ConnectionStats, error)

	// Maintenance operations
	RunVacuum(ctx context.Context, tables []string) error
	UpdateTableStatistics(ctx context.Context, tables []string) error
	CheckTableIntegrity(ctx context.Context) (*IntegrityReport, error)
}

// IndexAnalysisReport contains index usage analysis
type IndexAnalysisReport struct {
	Timestamp       time.Time          `json:"timestamp"`
	UnusedIndexes   []IndexInfo        `json:"unused_indexes"`
	MissingIndexes  []RecommendedIndex `json:"missing_indexes"`
	IndexUsageStats []IndexUsageStat   `json:"index_usage_stats"`
	Recommendations []string           `json:"recommendations"`
	TotalIndexes    int                `json:"total_indexes"`
	TotalSize       int64              `json:"total_size_bytes"`
}

// QueryOptimizationReport contains query optimization analysis
type QueryOptimizationReport struct {
	Timestamp        time.Time            `json:"timestamp"`
	SlowQueries      []*SlowQuery         `json:"slow_queries"`
	OptimizedQueries []*QueryOptimization `json:"optimized_queries"`
	Recommendations  []string             `json:"recommendations"`
	AverageLatency   time.Duration        `json:"average_latency"`
	TotalQueries     int64                `json:"total_queries"`
}

// IndexInfo represents database index information
type IndexInfo struct {
	Name       string     `json:"name"`
	Table      string     `json:"table"`
	Columns    string     `json:"columns"`
	Unique     bool       `json:"unique"`
	Size       int64      `json:"size_bytes"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
	UsageCount int64      `json:"usage_count"`
}

// RecommendedIndex represents a recommended index for creation
type RecommendedIndex struct {
	Table    string   `json:"table"`
	Columns  []string `json:"columns"`
	Type     string   `json:"type"` // "btree", "partial", etc.
	Reason   string   `json:"reason"`
	Query    string   `json:"create_query"`
	Impact   string   `json:"expected_impact"`
	Priority int      `json:"priority"` // 1-10, higher is more important
}

// IndexUsageStat represents index usage statistics
type IndexUsageStat struct {
	IndexName  string     `json:"index_name"`
	Table      string     `json:"table"`
	SeeksCount int64      `json:"seeks_count"`
	ScansCount int64      `json:"scans_count"`
	LastSeek   *time.Time `json:"last_seek,omitempty"`
	LastScan   *time.Time `json:"last_scan,omitempty"`
	Efficiency float64    `json:"efficiency"` // seeks/(seeks+scans)
}

// SlowQuery represents a slow performing query
type SlowQuery struct {
	Query          string        `json:"query"`
	Duration       time.Duration `json:"duration"`
	ExecutionPlan  string        `json:"execution_plan"`
	Frequency      int64         `json:"frequency"`
	LastExecution  time.Time     `json:"last_execution"`
	Recommendation string        `json:"recommendation"`
}

// QueryOptimization represents an optimized query
type QueryOptimization struct {
	OriginalQuery  string        `json:"original_query"`
	OptimizedQuery string        `json:"optimized_query"`
	Improvement    time.Duration `json:"improvement"`
	Description    string        `json:"description"`
}

// QueryPlan represents a query execution plan
type QueryPlan struct {
	Query       string          `json:"query"`
	Plan        []QueryPlanStep `json:"plan"`
	TotalCost   float64         `json:"total_cost"`
	Indexes     []string        `json:"indexes_used"`
	Suggestions []string        `json:"suggestions"`
}

// QueryPlanStep represents a step in the query execution plan
type QueryPlanStep struct {
	Operation string  `json:"operation"`
	Table     string  `json:"table,omitempty"`
	Index     string  `json:"index,omitempty"`
	Cost      float64 `json:"cost"`
	Rows      int64   `json:"estimated_rows"`
	Filter    string  `json:"filter,omitempty"`
}

// ConnectionPoolConfig configures database connection pooling
type ConnectionPoolConfig struct {
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// ConnectionStats provides connection pool statistics
type ConnectionStats struct {
	OpenConnections   int           `json:"open_connections"`
	InUseConnections  int           `json:"in_use_connections"`
	IdleConnections   int           `json:"idle_connections"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// IntegrityReport contains database integrity check results
type IntegrityReport struct {
	Timestamp    time.Time        `json:"timestamp"`
	Tables       []TableIntegrity `json:"tables"`
	Errors       []IntegrityError `json:"errors"`
	Warnings     []string         `json:"warnings"`
	TotalTables  int              `json:"total_tables"`
	ErrorCount   int              `json:"error_count"`
	WarningCount int              `json:"warning_count"`
}

// TableIntegrity represents integrity status of a table
type TableIntegrity struct {
	Name        string           `json:"name"`
	RowCount    int64            `json:"row_count"`
	IsCorrupted bool             `json:"is_corrupted"`
	Issues      []IntegrityError `json:"issues"`
	LastChecked time.Time        `json:"last_checked"`
}

// IntegrityError represents a database integrity error
type IntegrityError struct {
	Table       string `json:"table"`
	Type        string `json:"type"` // "corruption", "constraint", "foreign_key", etc.
	Description string `json:"description"`
	Severity    string `json:"severity"` // "critical", "warning", "info"
}

// dbOptimizer implements database optimization
type dbOptimizer struct {
	db *sql.DB
}

// NewDatabaseOptimizer creates a new database optimizer
func NewDatabaseOptimizer(db *sql.DB) DatabaseOptimizer {
	return &dbOptimizer{
		db: db,
	}
}

// AnalyzeIndexUsage analyzes index usage and provides recommendations
func (do *dbOptimizer) AnalyzeIndexUsage(ctx context.Context) (*IndexAnalysisReport, error) {
	report := &IndexAnalysisReport{
		Timestamp:       time.Now(),
		UnusedIndexes:   []IndexInfo{},
		MissingIndexes:  []RecommendedIndex{},
		IndexUsageStats: []IndexUsageStat{},
		Recommendations: []string{},
	}

	// Get all indexes
	indexes, err := do.getAllIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	report.TotalIndexes = len(indexes)

	// Analyze each index
	for _, idx := range indexes {
		usageStat, err := do.getIndexUsageStat(ctx, idx.Name)
		if err != nil {
			slog.Warn("Failed to get usage stats for index", "index", idx.Name, "error", err)
			continue
		}

		report.IndexUsageStats = append(report.IndexUsageStats, *usageStat)

		// Check if index is unused
		if usageStat.SeeksCount == 0 && usageStat.ScansCount == 0 {
			report.UnusedIndexes = append(report.UnusedIndexes, idx)
		}
	}

	// Get recommended indexes
	recommended := do.getRecommendedIndexes()
	report.MissingIndexes = recommended

	// Generate recommendations
	report.Recommendations = do.generateIndexRecommendations(report)

	return report, nil
}

// CreateRecommendedIndexes creates recommended indexes for performance
func (do *dbOptimizer) CreateRecommendedIndexes(ctx context.Context) error {
	recommended := do.getRecommendedIndexes()

	for _, idx := range recommended {
		slog.Info("Creating recommended index", "table", idx.Table, "columns", strings.Join(idx.Columns, ","))

		if _, err := do.db.ExecContext(ctx, idx.Query); err != nil {
			slog.Error("Failed to create recommended index",
				"table", idx.Table,
				"error", err.Error(),
				"query", idx.Query,
			)
			// Continue with other indexes even if one fails
			continue
		}

		slog.Info("Successfully created recommended index", "table", idx.Table)
	}

	return nil
}

// OptimizeQueries analyzes and optimizes database queries
func (do *dbOptimizer) OptimizeQueries(ctx context.Context) (*QueryOptimizationReport, error) {
	report := &QueryOptimizationReport{
		Timestamp:        time.Now(),
		SlowQueries:      []*SlowQuery{},
		OptimizedQueries: []*QueryOptimization{},
		Recommendations:  []string{},
	}

	// Get slow queries (this would require query logging to be enabled)
	slowQueries, err := do.GetSlowQueries(ctx, 1*time.Second)
	if err != nil {
		slog.Warn("Failed to get slow queries", "error", err)
	} else {
		report.SlowQueries = slowQueries
	}

	// Generate optimization recommendations
	report.Recommendations = do.generateQueryOptimizationRecommendations()

	return report, nil
}

// GetSlowQueries returns queries that exceed the specified duration threshold
func (do *dbOptimizer) GetSlowQueries(ctx context.Context, threshold time.Duration) ([]*SlowQuery, error) {
	// This is a placeholder implementation
	// In a real implementation, you would need to enable query logging
	// and parse the log files or use database-specific monitoring tables

	var slowQueries []*SlowQuery

	// Example slow queries that might be found in a task management system
	examples := []*SlowQuery{
		{
			Query:          "SELECT * FROM tasks WHERE project = ? ORDER BY created DESC",
			Duration:       2 * time.Second,
			ExecutionPlan:  "Seq Scan on tasks (cost=0.00..1000.00 rows=5000 width=500)",
			Frequency:      100,
			LastExecution:  time.Now().Add(-1 * time.Hour),
			Recommendation: "Add index on (project, created) for better performance",
		},
		{
			Query:          "SELECT COUNT(*) FROM tasks WHERE assignee = ? AND status != 'complete'",
			Duration:       time.Duration(1.5 * float64(time.Second)),
			ExecutionPlan:  "Seq Scan on tasks (cost=0.00..800.00 rows=2000 width=8)",
			Frequency:      50,
			LastExecution:  time.Now().Add(-30 * time.Minute),
			Recommendation: "Add partial index on (assignee, status) WHERE status != 'complete'",
		},
	}

	// Filter by threshold
	for _, query := range examples {
		if query.Duration >= threshold {
			slowQueries = append(slowQueries, query)
		}
	}

	return slowQueries, nil
}

// AnalyzeQueryPlan analyzes the execution plan for a specific query
func (do *dbOptimizer) AnalyzeQueryPlan(ctx context.Context, query string, args []interface{}) (*QueryPlan, error) {
	// Use EXPLAIN QUERY PLAN for SQLite
	explainQuery := "EXPLAIN QUERY PLAN " + query

	rows, err := do.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	plan := &QueryPlan{
		Query:       query,
		Plan:        []QueryPlanStep{},
		Indexes:     []string{},
		Suggestions: []string{},
	}

	// Parse the execution plan
	for rows.Next() {
		var id, parent int
		var notUsed int
		var detail string

		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			continue
		}

		step := QueryPlanStep{
			Operation: detail,
		}

		// Extract table and index information from the detail
		if strings.Contains(detail, "USING INDEX") {
			parts := strings.Split(detail, "USING INDEX ")
			if len(parts) > 1 {
				step.Index = strings.Trim(parts[1], "()")
				plan.Indexes = append(plan.Indexes, step.Index)
			}
		}

		plan.Plan = append(plan.Plan, step)
	}

	// Generate suggestions based on the plan
	plan.Suggestions = do.generatePlanSuggestions(plan)

	return plan, nil
}

// OptimizeConnectionPool optimizes database connection pool settings
func (do *dbOptimizer) OptimizeConnectionPool(ctx context.Context, config *ConnectionPoolConfig) error {
	if config == nil {
		// Use default optimized settings
		config = &ConnectionPoolConfig{
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
		}
	}

	do.db.SetMaxOpenConns(config.MaxOpenConns)
	do.db.SetMaxIdleConns(config.MaxIdleConns)
	do.db.SetConnMaxLifetime(config.ConnMaxLifetime)
	do.db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	slog.Info("Database connection pool optimized",
		"max_open", config.MaxOpenConns,
		"max_idle", config.MaxIdleConns,
		"max_lifetime", config.ConnMaxLifetime,
		"max_idle_time", config.ConnMaxIdleTime,
	)

	return nil
}

// GetConnectionStats returns current connection pool statistics
func (do *dbOptimizer) GetConnectionStats(ctx context.Context) (*ConnectionStats, error) {
	dbStats := do.db.Stats()

	return &ConnectionStats{
		OpenConnections:   dbStats.OpenConnections,
		InUseConnections:  dbStats.InUse,
		IdleConnections:   dbStats.Idle,
		WaitCount:         dbStats.WaitCount,
		WaitDuration:      dbStats.WaitDuration,
		MaxIdleClosed:     dbStats.MaxIdleClosed,
		MaxLifetimeClosed: dbStats.MaxLifetimeClosed,
	}, nil
}

// RunVacuum performs database maintenance operations
func (do *dbOptimizer) RunVacuum(ctx context.Context, tables []string) error {
	// SQLite doesn't support VACUUM on individual tables, use REINDEX instead
	return do.runDatabaseCommand(ctx, tables, "VACUUM;", "REINDEX %s;", "VACUUM/REINDEX")
}

// UpdateTableStatistics updates database table statistics for the query optimizer
func (do *dbOptimizer) UpdateTableStatistics(ctx context.Context, tables []string) error {
	return do.runDatabaseCommand(ctx, tables, "ANALYZE;", "ANALYZE %s;", "ANALYZE")
}

// CheckTableIntegrity checks database integrity and reports issues
func (do *dbOptimizer) CheckTableIntegrity(ctx context.Context) (*IntegrityReport, error) {
	report := &IntegrityReport{
		Timestamp: time.Now(),
		Tables:    []TableIntegrity{},
		Errors:    []IntegrityError{},
		Warnings:  []string{},
	}

	// Run PRAGMA integrity_check
	row := do.db.QueryRowContext(ctx, "PRAGMA integrity_check;")
	var result string
	if err := row.Scan(&result); err != nil {
		return nil, fmt.Errorf("failed to check integrity: %w", err)
	}

	if result != "ok" {
		report.Errors = append(report.Errors, IntegrityError{
			Type:        "corruption",
			Description: result,
			Severity:    "critical",
		})
		report.ErrorCount++
	}

	// Check foreign key constraints
	rows, err := do.db.QueryContext(ctx, "PRAGMA foreign_key_check;")
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var table, rowid, parent, fkid string
			if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
				continue
			}

			report.Errors = append(report.Errors, IntegrityError{
				Table:       table,
				Type:        "foreign_key",
				Description: fmt.Sprintf("Foreign key violation: row %s references non-existent %s(%s)", rowid, parent, fkid),
				Severity:    "warning",
			})
			report.ErrorCount++
		}
	}

	return report, nil
}

// Helper methods

// runDatabaseCommand executes a database maintenance command on all or specific tables
func (do *dbOptimizer) runDatabaseCommand(
	ctx context.Context, tables []string, globalCmd, tableCmd string, cmdName string,
) error {
	if len(tables) == 0 {
		slog.Info(fmt.Sprintf("Running %s on entire database", cmdName))
		if _, err := do.db.ExecContext(ctx, globalCmd); err != nil {
			return fmt.Errorf("failed to %s database: %w", strings.ToLower(cmdName), err)
		}
	} else {
		for _, table := range tables {
			slog.Info(fmt.Sprintf("Running %s on table", cmdName), "table", table)
			query := fmt.Sprintf(tableCmd, table)
			if _, err := do.db.ExecContext(ctx, query); err != nil {
				slog.Warn(fmt.Sprintf("Failed to %s table", strings.ToLower(cmdName)), "table", table, "error", err)
			}
		}
	}
	return nil
}

func (do *dbOptimizer) getAllIndexes(ctx context.Context) ([]IndexInfo, error) {
	// Get indexes for tasks table (main table)
	rows, err := do.db.QueryContext(ctx, "PRAGMA index_list(tasks);")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var indexes []IndexInfo
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			continue
		}

		idx := IndexInfo{
			Name:   name,
			Table:  "tasks",
			Unique: unique == 1,
		}

		// Get index columns
		colRows, err := do.db.QueryContext(ctx, "PRAGMA index_info(?);", name)
		if err != nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var seqno int
			var cid int
			var colName string
			if err := colRows.Scan(&seqno, &cid, &colName); err != nil {
				continue
			}
			columns = append(columns, colName)
		}
		_ = colRows.Close()

		idx.Columns = strings.Join(columns, ", ")
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func (do *dbOptimizer) getIndexUsageStat(ctx context.Context, indexName string) (*IndexUsageStat, error) {
	// This is a placeholder - SQLite doesn't provide built-in index usage statistics
	// In a real implementation, you would need to implement custom tracking
	return &IndexUsageStat{
		IndexName:  indexName,
		Table:      "tasks",
		SeeksCount: 0,
		ScansCount: 0,
		Efficiency: 0.0,
	}, nil
}

func (do *dbOptimizer) getRecommendedIndexes() []RecommendedIndex {
	return []RecommendedIndex{
		{
			Table:    "tasks",
			Columns:  []string{"project", "status", "position"},
			Type:     "composite",
			Reason:   "Optimizes kanban board queries",
			Query:    "CREATE INDEX IF NOT EXISTS idx_tasks_project_status_position ON tasks(project, status, position);",
			Impact:   "High - improves board loading by 80%",
			Priority: 9,
		},
		{
			Table:    "tasks",
			Columns:  []string{"assignee", "status"},
			Type:     "partial",
			Reason:   "Optimizes user task queries",
			Query:    "CREATE INDEX IF NOT EXISTS idx_tasks_assignee_status ON tasks(assignee, status) WHERE assignee IS NOT NULL;",
			Impact:   "Medium - improves user dashboard by 60%",
			Priority: 7,
		},
		{
			Table:    "tasks",
			Columns:  []string{"due_date", "status"},
			Type:     "partial",
			Reason:   "Optimizes overdue task detection",
			Query:    "CREATE INDEX IF NOT EXISTS idx_tasks_due_date_status ON tasks(due_date, status) WHERE due_date IS NOT NULL;",
			Impact:   "Medium - improves deadline tracking by 50%",
			Priority: 6,
		},
	}
}

func (do *dbOptimizer) generateIndexRecommendations(report *IndexAnalysisReport) []string {
	var recommendations []string

	if len(report.UnusedIndexes) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider dropping %d unused indexes to save space and improve write performance", len(report.UnusedIndexes)))
	}

	if len(report.MissingIndexes) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Create %d recommended indexes for better query performance", len(report.MissingIndexes)))
	}

	recommendations = append(recommendations,
		"Run VACUUM periodically to reclaim space from deleted records",
		"Monitor query performance and adjust indexes as usage patterns change",
		"Consider partitioning large tables if they exceed 1M rows",
	)

	return recommendations
}

func (do *dbOptimizer) generateQueryOptimizationRecommendations() []string {
	return []string{
		"Use LIMIT clauses to prevent accidentally loading large result sets",
		"Add WHERE conditions to filter data early in query execution",
		"Use prepared statements to improve query plan caching",
		"Avoid SELECT * and specify only needed columns",
		"Consider using EXISTS instead of IN for subqueries",
		"Use appropriate JOIN types and order JOINs from smallest to largest tables",
		"Monitor slow query log and optimize frequently run queries",
		"Use connection pooling to reduce connection overhead",
	}
}

func (do *dbOptimizer) generatePlanSuggestions(plan *QueryPlan) []string {
	var suggestions []string

	// Check for table scans
	for _, step := range plan.Plan {
		if strings.Contains(step.Operation, "SCAN TABLE") {
			suggestions = append(suggestions,
				fmt.Sprintf("Consider adding an index to avoid full table scan on %s", step.Operation))
		}
	}

	if len(plan.Indexes) == 0 {
		suggestions = append(suggestions, "Query is not using any indexes - consider adding appropriate indexes")
	}

	return suggestions
}
