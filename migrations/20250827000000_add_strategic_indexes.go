package migrations

import (
	"database/sql"
	"log/slog"

	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db *sql.DB) error {
		slog.Info("Starting strategic index creation for performance optimization")
		
		// Strategic indexes for task queries based on Week 8 requirements
		indexes := []struct {
			name  string
			table string
			query string
			desc  string
		}{
			{
				name:  "idx_tasks_project_status_position",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_project_status_position ON tasks(project, status, position);",
				desc:  "Optimizes kanban board queries by project and status with position ordering",
			},
			{
				name:  "idx_tasks_assignee_status",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_assignee_status ON tasks(assignee, status) WHERE assignee IS NOT NULL;",
				desc:  "Optimizes user task queries filtered by assignment and status",
			},
			{
				name:  "idx_tasks_due_date_status",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_due_date_status ON tasks(due_date, status) WHERE due_date IS NOT NULL;",
				desc:  "Optimizes overdue task queries and due date filtering",
			},
			{
				name:  "idx_tasks_reporter_created",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_reporter_created ON tasks(reporter, created);",
				desc:  "Optimizes queries for tasks created by specific users",
			},
			{
				name:  "idx_tasks_parent_task",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_parent_task ON tasks(parent_task) WHERE parent_task IS NOT NULL;",
				desc:  "Optimizes subtask lookups and hierarchical task queries",
			},
			{
				name:  "idx_tasks_project_updated",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_project_updated ON tasks(project, updated);",
				desc:  "Optimizes recently updated task queries per project",
			},
			{
				name:  "idx_tasks_priority_status",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_priority_status ON tasks(priority, status);",
				desc:  "Optimizes priority-based filtering and sorting",
			},
			{
				name:  "idx_tasks_archived_project",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_archived_project ON tasks(archived, project);",
				desc:  "Optimizes queries that exclude archived tasks",
			},
			
			// Indexes for supporting collections
			{
				name:  "idx_comments_task_created",
				table: "comments",
				query: "CREATE INDEX IF NOT EXISTS idx_comments_task_created ON comments(task, created);",
				desc:  "Optimizes comment retrieval for tasks ordered by creation date",
			},
			{
				name:  "idx_task_history_task_created",
				table: "task_history",
				query: "CREATE INDEX IF NOT EXISTS idx_task_history_task_created ON task_history(task, created);",
				desc:  "Optimizes task history queries ordered by timestamp",
			},
			{
				name:  "idx_task_history_user_action",
				table: "task_history",
				query: "CREATE INDEX IF NOT EXISTS idx_task_history_user_action ON task_history(user, action, created);",
				desc:  "Optimizes audit queries by user and action type",
			},
			{
				name:  "idx_tags_project_usage",
				table: "tags",
				query: "CREATE INDEX IF NOT EXISTS idx_tags_project_usage ON tags(project, usage_count);",
				desc:  "Optimizes tag queries by project ordered by usage count",
			},
			
			// Composite indexes for complex queries
			{
				name:  "idx_tasks_complex_kanban",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_complex_kanban ON tasks(project, status, priority, position) WHERE archived = false;",
				desc:  "Comprehensive index for kanban board queries with priority filtering",
			},
			{
				name:  "idx_tasks_search_optimization",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_search_optimization ON tasks(project, archived, status, updated);",
				desc:  "Optimizes search and filtering operations",
			},
			{
				name:  "idx_tasks_assignment_workload",
				table: "tasks",
				query: "CREATE INDEX IF NOT EXISTS idx_tasks_assignment_workload ON tasks(assignee, status, priority) WHERE assignee IS NOT NULL AND archived = false;",
				desc:  "Optimizes workload queries for assigned tasks",
			},
		}

		// Create indexes
		for _, idx := range indexes {
			slog.Info("Creating index", "name", idx.name, "table", idx.table, "description", idx.desc)
			
			if _, err := db.Exec(idx.query); err != nil {
				slog.Error("Failed to create index", 
					"name", idx.name, 
					"table", idx.table, 
					"error", err.Error(),
					"query", idx.query,
				)
				return err
			}
			
			slog.Info("Successfully created index", "name", idx.name, "table", idx.table)
		}

		slog.Info("Strategic indexes created successfully")
		return nil
	}, func(db *sql.DB) error {
		// Rollback: Drop the created indexes
		slog.Info("Rolling back strategic indexes")
		
		indexes := []string{
			"DROP INDEX IF EXISTS idx_tasks_project_status_position;",
			"DROP INDEX IF EXISTS idx_tasks_assignee_status;",
			"DROP INDEX IF EXISTS idx_tasks_due_date_status;",
			"DROP INDEX IF EXISTS idx_tasks_reporter_created;",
			"DROP INDEX IF EXISTS idx_tasks_parent_task;",
			"DROP INDEX IF EXISTS idx_tasks_project_updated;",
			"DROP INDEX IF EXISTS idx_tasks_priority_status;",
			"DROP INDEX IF EXISTS idx_tasks_archived_project;",
			"DROP INDEX IF EXISTS idx_comments_task_created;",
			"DROP INDEX IF EXISTS idx_task_history_task_created;",
			"DROP INDEX IF EXISTS idx_task_history_user_action;",
			"DROP INDEX IF EXISTS idx_tags_project_usage;",
			"DROP INDEX IF EXISTS idx_tasks_complex_kanban;",
			"DROP INDEX IF EXISTS idx_tasks_search_optimization;",
			"DROP INDEX IF EXISTS idx_tasks_assignment_workload;",
		}

		for _, query := range indexes {
			if _, err := db.Exec(query); err != nil {
				slog.Warn("Failed to drop index during rollback", "error", err.Error(), "query", query)
				// Continue with other indexes even if one fails
			}
		}

		slog.Info("Strategic indexes rollback completed")
		return nil
	})
}

// GetRecommendedIndexes returns a list of recommended indexes for query optimization
func GetRecommendedIndexes() map[string]string {
	return map[string]string{
		"kanban_board_queries": `
			-- Primary kanban board query optimization
			CREATE INDEX IF NOT EXISTS idx_tasks_project_status_position 
			ON tasks(project, status, position);
		`,
		"user_assignments": `
			-- User task assignment queries
			CREATE INDEX IF NOT EXISTS idx_tasks_assignee_status 
			ON tasks(assignee, status) WHERE assignee IS NOT NULL;
		`,
		"due_date_tracking": `
			-- Overdue and due date filtering
			CREATE INDEX IF NOT EXISTS idx_tasks_due_date_status 
			ON tasks(due_date, status) WHERE due_date IS NOT NULL;
		`,
		"subtask_hierarchies": `
			-- Subtask and parent-child relationships
			CREATE INDEX IF NOT EXISTS idx_tasks_parent_task 
			ON tasks(parent_task) WHERE parent_task IS NOT NULL;
		`,
		"audit_trail": `
			-- Task history and audit logging
			CREATE INDEX IF NOT EXISTS idx_task_history_task_created 
			ON task_history(task, created);
		`,
		"search_optimization": `
			-- General search and filtering
			CREATE INDEX IF NOT EXISTS idx_tasks_search_optimization 
			ON tasks(project, archived, status, updated);
		`,
	}
}

// AnalyzeQueryPerformance provides SQL for analyzing query performance
func AnalyzeQueryPerformance() []string {
	return []string{
		`-- Analyze most common task queries
		EXPLAIN QUERY PLAN 
		SELECT * FROM tasks 
		WHERE project = ? AND status = ? 
		ORDER BY position;`,
		
		`-- Analyze user assignment queries  
		EXPLAIN QUERY PLAN
		SELECT * FROM tasks 
		WHERE assignee = ? AND archived = false
		ORDER BY priority, due_date;`,
		
		`-- Analyze due date queries
		EXPLAIN QUERY PLAN
		SELECT * FROM tasks 
		WHERE due_date < datetime('now') AND status != 'complete'
		ORDER BY due_date;`,
		
		`-- Analyze subtask queries
		EXPLAIN QUERY PLAN
		SELECT * FROM tasks 
		WHERE parent_task = ?
		ORDER BY position;`,
		
		`-- Analyze search queries
		EXPLAIN QUERY PLAN
		SELECT * FROM tasks 
		WHERE project = ? AND archived = false
		AND (title LIKE ? OR description LIKE ?)
		ORDER BY updated DESC;`,
	}
}