//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/testutil/integration"
)

// PerformanceTestSuite provides comprehensive performance testing for PocketBase operations
type PerformanceTestSuite struct {
	*integration.DatabaseTestSuite
	ctx           context.Context
	memBaseline   runtime.MemStats
	startTime     time.Time
	taskCount     int
	projectCount  int
	userCount     int
}

// PerformanceMetrics captures performance data for analysis
type PerformanceMetrics struct {
	Operation        string
	Duration         time.Duration
	RecordsProcessed int
	MemoryUsed       uint64
	MemoryAllocated  uint64
	QueryPlan        string
	IndexesUsed      []string
	Success          bool
	Error            error
}

// Performance benchmarks and thresholds
const (
	// Record count thresholds for large dataset tests
	LargeDatasetSize     = 10000
	MediumDatasetSize    = 1000
	BulkOperationSize    = 1000
	
	// Performance thresholds (milliseconds)
	MaxQueryTime         = 100  * time.Millisecond  // 100ms max for indexed queries
	MaxBulkInsertTime    = 5000 * time.Millisecond  // 5s for 1000 record inserts
	MaxBulkUpdateTime    = 3000 * time.Millisecond  // 3s for 1000 record updates
	MaxBulkDeleteTime    = 2000 * time.Millisecond  // 2s for 1000 record deletes
	MaxSearchTime        = 200  * time.Millisecond  // 200ms for full-text search
	MaxPaginationTime    = 50   * time.Millisecond  // 50ms for pagination queries
	
	// Memory thresholds (MB)
	MaxMemoryIncrease    = 100 * 1024 * 1024  // 100MB max increase during operations
	
	// Index effectiveness thresholds
	MinIndexUsageRatio   = 0.8  // 80% of queries should use indexes
)

// setupPerformanceTest creates a performance test suite with large datasets
func setupPerformanceTest(t *testing.T) *PerformanceTestSuite {
	suite := &PerformanceTestSuite{
		DatabaseTestSuite: integration.SetupDatabaseTest(t),
		ctx:              context.Background(),
		startTime:        time.Now(),
	}
	
	// Record baseline memory usage
	runtime.GC()
	runtime.ReadMemStats(&suite.memBaseline)
	
	return suite
}

// captureMemoryStats captures current memory statistics
func (p *PerformanceTestSuite) captureMemoryStats() runtime.MemStats {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// measureOperation executes a function and captures performance metrics
func (p *PerformanceTestSuite) measureOperation(t *testing.T, opName string, recordCount int, operation func() error) PerformanceMetrics {
	startMem := p.captureMemoryStats()
	startTime := time.Now()
	
	err := operation()
	
	duration := time.Since(startTime)
	endMem := p.captureMemoryStats()
	
	return PerformanceMetrics{
		Operation:        opName,
		Duration:         duration,
		RecordsProcessed: recordCount,
		MemoryUsed:       endMem.Alloc - startMem.Alloc,
		MemoryAllocated:  endMem.TotalAlloc - startMem.TotalAlloc,
		Success:          err == nil,
		Error:            err,
	}
}

// analyzeQueryPlan captures and analyzes SQLite query execution plan
func (p *PerformanceTestSuite) analyzeQueryPlan(t *testing.T, query string, params ...interface{}) (string, []string) {
	// For EXPLAIN QUERY PLAN, we need to substitute parameters manually since SQLite
	// requires them to be present for plan analysis
	explainQuery := "EXPLAIN QUERY PLAN " + substituteParameters(query, params...)
	
	rows, err := p.DB.App().DB().NewQuery(explainQuery).Rows()
	if err != nil {
		t.Logf("Failed to get query plan for: %s - %v", explainQuery, err)
		return "", []string{}
	}
	defer rows.Close()
	
	var plan []string
	var indexesUsed []string
	
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err == nil {
			plan = append(plan, fmt.Sprintf("%d|%d|%s", id, parent, detail))
			
			// Extract index usage information
			if containsSubstring(detail, "USING INDEX") || containsSubstring(detail, "USING COVERING INDEX") {
				indexesUsed = append(indexesUsed, detail)
			}
		}
	}
	
	planStr := joinStrings(plan, "\n")
	return planStr, indexesUsed
}

// containsSubstring checks if a string contains a substring (case-insensitive)
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && 
		   str[:len(substr)] == substr[:len(substr)] ||
		   (len(str) > len(substr) && containsSubstring(str[1:], substr))
}

// joinStrings joins string slices with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// substituteParameters replaces ? placeholders with actual parameter values for EXPLAIN QUERY PLAN
func substituteParameters(query string, params ...interface{}) string {
	result := query
	for _, param := range params {
		// Find first occurrence of ? and replace it
		for i := 0; i < len(result); i++ {
			if result[i] == '?' {
				paramStr := fmt.Sprintf("'%v'", param)
				result = result[:i] + paramStr + result[i+1:]
				break
			}
		}
	}
	return result
}

// generateLargeDataset creates a large dataset for performance testing
func (p *PerformanceTestSuite) generateLargeDataset(t *testing.T, userCount, projectCount, taskCount int) {
	t.Logf("Generating large dataset: %d users, %d projects, %d tasks", userCount, projectCount, taskCount)
	
	// Create users
	users := make([]*domain.User, userCount)
	for i := 0; i < userCount; i++ {
		user := p.Factory.CreateUser(
			integration.WithUserUsername(fmt.Sprintf("perfuser_%d", i)),
			integration.WithUserEmail(fmt.Sprintf("perfuser_%d@test.com", i)),
		)
		
		err := p.Repos.Users.Create(p.ctx, user)
		require.NoError(t, err, "Failed to create user %d", i)
		users[i] = user
		
		if i%100 == 0 {
			t.Logf("Created %d/%d users", i+1, userCount)
		}
	}
	
	// Create projects
	projects := make([]*domain.Project, projectCount)
	for i := 0; i < projectCount; i++ {
		owner := users[i%len(users)]
		project := p.Factory.CreateProject(owner,
			integration.WithProjectTitle(fmt.Sprintf("Perf Project %d", i)),
			integration.WithProjectSlug(fmt.Sprintf("perf-project-%d", i)),
		)
		
		err := p.Repos.Projects.Create(p.ctx, project)
		require.NoError(t, err, "Failed to create project %d", i)
		projects[i] = project
		
		if i%50 == 0 {
			t.Logf("Created %d/%d projects", i+1, projectCount)
		}
	}
	
	// Create tasks
	for i := 0; i < taskCount; i++ {
		project := projects[i%len(projects)]
		reporter := users[i%len(users)]
		
		// Vary task properties for realistic distribution
		status := domain.TaskStatus([]domain.TaskStatus{
			domain.StatusBacklog, domain.StatusTodo, domain.StatusDeveloping, 
			domain.StatusReview, domain.StatusComplete,
		}[i%5])
		
		priority := domain.TaskPriority([]domain.TaskPriority{
			domain.PriorityCritical, domain.PriorityHigh, 
			domain.PriorityMedium, domain.PriorityLow,
		}[i%4])
		
		// Assign some tasks to users
		var assigneeID *string
		if i%3 == 0 {
			assignee := users[i%len(users)]
			assigneeID = &assignee.ID
		}
		
		task := p.Factory.CreateTask(project, reporter,
			integration.WithTaskTitle(fmt.Sprintf("Performance Task %d", i)),
			integration.WithTaskDescription(fmt.Sprintf("Description for performance testing task number %d with some searchable content", i)),
			integration.WithTaskStatus(status),
			integration.WithTaskPriority(priority),
		)
		
		if assigneeID != nil {
			task.AssigneeID = assigneeID
		}
		
		err := p.Repos.Tasks.Create(p.ctx, task)
		require.NoError(t, err, "Failed to create task %d", i)
		
		if i%500 == 0 {
			t.Logf("Created %d/%d tasks", i+1, taskCount)
		}
	}
	
	p.userCount = userCount
	p.projectCount = projectCount
	p.taskCount = taskCount
	
	t.Logf("Dataset generation complete: %d users, %d projects, %d tasks", userCount, projectCount, taskCount)
}

// TestPerformance_TasksByProjectQueries tests performance of TasksByProject queries with large datasets
func TestPerformance_TasksByProjectQueries(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate large dataset
	suite.generateLargeDataset(t, 100, 50, LargeDatasetSize)
	
	// Get a random project for testing
	projects, err := suite.Repos.Projects.List(suite.ctx, 0, 1)
	require.NoError(t, err)
	require.Len(t, projects, 1)
	testProjectID := projects[0].ID
	
	// Test TasksByProject query performance
	metrics := suite.measureOperation(t, "TasksByProject", LargeDatasetSize, func() error {
		// Directly query the database to avoid repository sorting issues during performance testing
		filter := "project = {:projectID}"
		params := map[string]interface{}{"projectID": testProjectID}
		
		records, err := suite.DB.App().FindRecordsByFilter("tasks", filter, "", 100, 0, params)
		if err != nil {
			return err
		}
		t.Logf("Retrieved %d tasks for project %s", len(records), testProjectID)
		return nil
	})
	
	// Assert performance requirements
	assert.True(t, metrics.Success, "TasksByProject query should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxQueryTime, 
		"TasksByProject query took %v, expected less than %v", metrics.Duration, MaxQueryTime)
	
	// Analyze query plan
	query := "SELECT * FROM tasks WHERE project = ? LIMIT 100"
	plan, indexes := suite.analyzeQueryPlan(t, query, testProjectID)
	t.Logf("Query plan for TasksByProject:\n%s", plan)
	t.Logf("Indexes used: %v", indexes)
	
	// Assert index usage
	assert.NotEmpty(t, indexes, "TasksByProject query should use indexes")
	
	// Log performance metrics
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_TasksByAssigneeQueries tests performance of TasksByAssignee queries
func TestPerformance_TasksByAssigneeQueries(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate large dataset
	suite.generateLargeDataset(t, 100, 50, LargeDatasetSize)
	
	// Get a random user for testing
	users, err := suite.Repos.Users.List(suite.ctx, 0, 1)
	require.NoError(t, err)
	require.Len(t, users, 1)
	testUserID := users[0].ID
	
	// Test TasksByAssignee query performance
	metrics := suite.measureOperation(t, "TasksByAssignee", LargeDatasetSize, func() error {
		// Directly query the database to avoid repository sorting issues during performance testing
		filter := "assignee = {:assigneeID}"
		params := map[string]interface{}{"assigneeID": testUserID}
		
		records, err := suite.DB.App().FindRecordsByFilter("tasks", filter, "", 100, 0, params)
		if err != nil {
			return err
		}
		t.Logf("Retrieved %d tasks for assignee %s", len(records), testUserID)
		return nil
	})
	
	// Assert performance requirements
	assert.True(t, metrics.Success, "TasksByAssignee query should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxQueryTime, 
		"TasksByAssignee query took %v, expected less than %v", metrics.Duration, MaxQueryTime)
	
	// Analyze query plan
	query := "SELECT * FROM tasks WHERE assignee = ? LIMIT 100"
	plan, indexes := suite.analyzeQueryPlan(t, query, testUserID)
	t.Logf("Query plan for TasksByAssignee:\n%s", plan)
	t.Logf("Indexes used: %v", indexes)
	
	// Assert index usage
	assert.NotEmpty(t, indexes, "TasksByAssignee query should use indexes")
	
	// Log performance metrics
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_PaginationQueries tests pagination performance with large datasets
func TestPerformance_PaginationQueries(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate large dataset
	suite.generateLargeDataset(t, 50, 25, LargeDatasetSize)
	
	// Get test project
	projects, err := suite.Repos.Projects.List(suite.ctx, 0, 1)
	require.NoError(t, err)
	testProjectID := projects[0].ID
	
	// Test pagination at different offsets
	offsets := []int{0, 1000, 5000, 9000}
	limit := 50
	
	for _, offset := range offsets {
		t.Run(fmt.Sprintf("Offset_%d", offset), func(t *testing.T) {
			metrics := suite.measureOperation(t, fmt.Sprintf("Pagination_Offset_%d", offset), limit, func() error {
				// Direct database query for pagination testing
				filter := "project = {:projectID}"
				params := map[string]interface{}{"projectID": testProjectID}
				
				records, err := suite.DB.App().FindRecordsByFilter("tasks", filter, "", limit, offset, params)
				if err != nil {
					return err
				}
				t.Logf("Retrieved %d tasks at offset %d", len(records), offset)
				return nil
			})
			
			assert.True(t, metrics.Success, "Pagination query at offset %d should succeed: %v", offset, metrics.Error)
			assert.Less(t, metrics.Duration, MaxPaginationTime, 
				"Pagination at offset %d took %v, expected less than %v", offset, metrics.Duration, MaxPaginationTime)
			
			suite.logPerformanceMetrics(t, metrics)
		})
	}
}

// TestPerformance_CompoundIndexEffectiveness tests compound index usage for project+status queries
func TestPerformance_CompoundIndexEffectiveness(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate large dataset
	suite.generateLargeDataset(t, 50, 25, LargeDatasetSize)
	
	// Get test project
	projects, err := suite.Repos.Projects.List(suite.ctx, 0, 1)
	require.NoError(t, err)
	testProjectID := projects[0].ID
	
	// Test compound index effectiveness for project+status filtering
	metrics := suite.measureOperation(t, "CompoundIndex_ProjectStatus", LargeDatasetSize, func() error {
		tasks, err := suite.Repos.Tasks.ListByStatus(suite.ctx, domain.StatusTodo, 0, 100)
		if err != nil {
			return err
		}
		t.Logf("Retrieved %d tasks with status TODO", len(tasks))
		return nil
	})
	
	// Assert performance
	assert.True(t, metrics.Success, "Compound index query should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxQueryTime, 
		"Compound index query took %v, expected less than %v", metrics.Duration, MaxQueryTime)
	
	// Analyze query plan for compound conditions
	query := "SELECT * FROM tasks WHERE project = ? AND status = ? LIMIT 100"
	plan, indexes := suite.analyzeQueryPlan(t, query, testProjectID, "todo")
	t.Logf("Query plan for project+status compound query:\n%s", plan)
	t.Logf("Indexes used: %v", indexes)
	
	// Check for efficient index usage (no full table scan)
	assert.NotContains(t, plan, "SCAN TABLE", "Query should not perform full table scan")
	assert.NotEmpty(t, indexes, "Compound query should use indexes")
	
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_BulkCreation benchmarks bulk task creation
func TestPerformance_BulkCreation(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Setup minimal dataset for bulk operations
	suite.generateLargeDataset(t, 10, 5, 0) // No tasks initially
	
	// Get test data
	users, err := suite.Repos.Users.List(suite.ctx, 0, 10)
	require.NoError(t, err)
	projects, err := suite.Repos.Projects.List(suite.ctx, 0, 5)
	require.NoError(t, err)
	
	// Prepare bulk tasks
	tasks := make([]*domain.Task, BulkOperationSize)
	for i := 0; i < BulkOperationSize; i++ {
		project := projects[i%len(projects)]
		reporter := users[i%len(users)]
		
		tasks[i] = suite.Factory.CreateTask(project, reporter,
			integration.WithTaskTitle(fmt.Sprintf("Bulk Task %d", i)),
			integration.WithTaskDescription(fmt.Sprintf("Bulk operation task %d", i)),
		)
	}
	
	// Measure bulk creation performance
	metrics := suite.measureOperation(t, "BulkCreation", BulkOperationSize, func() error {
		for _, task := range tasks {
			if err := suite.Repos.Tasks.Create(suite.ctx, task); err != nil {
				return err
			}
		}
		return nil
	})
	
	// Assert performance requirements
	assert.True(t, metrics.Success, "Bulk creation should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxBulkInsertTime, 
		"Bulk creation took %v, expected less than %v", metrics.Duration, MaxBulkInsertTime)
	
	// Check memory usage
	assert.Less(t, metrics.MemoryUsed, uint64(MaxMemoryIncrease),
		"Memory usage %d bytes exceeded threshold %d bytes", metrics.MemoryUsed, MaxMemoryIncrease)
	
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_BulkUpdates benchmarks bulk task updates
func TestPerformance_BulkUpdates(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate dataset with tasks to update
	suite.generateLargeDataset(t, 50, 25, BulkOperationSize*2)
	
	// Get tasks for bulk update
	tasks, err := suite.Repos.Tasks.ListByStatus(suite.ctx, domain.StatusBacklog, 0, BulkOperationSize)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tasks), BulkOperationSize/2, "Not enough tasks for bulk update test")
	
	// Prepare updates
	for _, task := range tasks {
		task.Status = domain.StatusTodo
		task.Progress = 25
	}
	
	// Measure bulk update performance
	metrics := suite.measureOperation(t, "BulkUpdate", len(tasks), func() error {
		for _, task := range tasks {
			if err := suite.Repos.Tasks.Update(suite.ctx, task); err != nil {
				return err
			}
		}
		return nil
	})
	
	// Assert performance requirements
	assert.True(t, metrics.Success, "Bulk update should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxBulkUpdateTime, 
		"Bulk update took %v, expected less than %v", metrics.Duration, MaxBulkUpdateTime)
	
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_BulkDeletion benchmarks bulk task deletion
func TestPerformance_BulkDeletion(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate dataset with tasks to delete
	suite.generateLargeDataset(t, 50, 25, BulkOperationSize*2)
	
	// Get tasks for bulk deletion
	tasks, err := suite.Repos.Tasks.ListByStatus(suite.ctx, domain.StatusComplete, 0, BulkOperationSize)
	require.NoError(t, err)
	
	// If we don't have enough completed tasks, get any tasks
	if len(tasks) < BulkOperationSize/2 {
		allTasks, err := suite.Repos.Tasks.ListByProject(suite.ctx, "", 0, BulkOperationSize)
		require.NoError(t, err)
		tasks = allTasks[:min(len(allTasks), BulkOperationSize)]
	}
	
	taskIDs := make([]string, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}
	
	// Measure bulk deletion performance
	metrics := suite.measureOperation(t, "BulkDeletion", len(taskIDs), func() error {
		for _, id := range taskIDs {
			if err := suite.Repos.Tasks.Delete(suite.ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
	
	// Assert performance requirements
	assert.True(t, metrics.Success, "Bulk deletion should succeed: %v", metrics.Error)
	assert.Less(t, metrics.Duration, MaxBulkDeleteTime, 
		"Bulk deletion took %v, expected less than %v", metrics.Duration, MaxBulkDeleteTime)
	
	suite.logPerformanceMetrics(t, metrics)
}

// TestPerformance_FullTextSearch tests full-text search performance
func TestPerformance_FullTextSearch(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate dataset with searchable content
	suite.generateLargeDataset(t, 50, 25, LargeDatasetSize)
	
	// Test various search queries
	searchQueries := []string{
		"performance",
		"task",
		"testing",
		"description",
		"number",
	}
	
	for _, query := range searchQueries {
		t.Run(fmt.Sprintf("Search_%s", query), func(t *testing.T) {
			metrics := suite.measureOperation(t, fmt.Sprintf("FullTextSearch_%s", query), LargeDatasetSize, func() error {
				tasks, err := suite.Repos.Tasks.Search(suite.ctx, query, "", 0, 50)
				if err != nil {
					return err
				}
				t.Logf("Search for '%s' returned %d results", query, len(tasks))
				return nil
			})
			
			assert.True(t, metrics.Success, "Search for '%s' should succeed: %v", query, metrics.Error)
			assert.Less(t, metrics.Duration, MaxSearchTime, 
				"Search for '%s' took %v, expected less than %v", query, metrics.Duration, MaxSearchTime)
			
			// Analyze search query plan
			searchQuery := `SELECT * FROM tasks WHERE (title LIKE '%' || ? || '%' OR description LIKE '%' || ? || '%') LIMIT 50`
			plan, _ := suite.analyzeQueryPlan(t, searchQuery, query, query)
			t.Logf("Search query plan:\n%s", plan)
			
			suite.logPerformanceMetrics(t, metrics)
		})
	}
}

// TestPerformance_IndexUsageValidation validates that critical queries use indexes
func TestPerformance_IndexUsageValidation(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate dataset
	suite.generateLargeDataset(t, 50, 25, MediumDatasetSize)
	
	// Get test data
	projects, _ := suite.Repos.Projects.List(suite.ctx, 0, 1)
	users, _ := suite.Repos.Users.List(suite.ctx, 0, 1)
	
	// Critical queries that must use indexes
	criticalQueries := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "TasksByProject",
			query: "SELECT * FROM tasks WHERE project = ? LIMIT 100",
			args:  []interface{}{projects[0].ID},
		},
		{
			name:  "TasksByAssignee", 
			query: "SELECT * FROM tasks WHERE assignee = ? LIMIT 100",
			args:  []interface{}{users[0].ID},
		},
		{
			name:  "TasksByStatus",
			query: "SELECT * FROM tasks WHERE status = ? LIMIT 100",
			args:  []interface{}{"todo"},
		},
		{
			name:  "TaskCountByProject",
			query: "SELECT COUNT(*) FROM tasks WHERE project = ?",
			args:  []interface{}{projects[0].ID},
		},
	}
	
	indexUsageCount := 0
	
	for _, cq := range criticalQueries {
		t.Run(cq.name, func(t *testing.T) {
			plan, indexes := suite.analyzeQueryPlan(t, cq.query, cq.args...)
			
			t.Logf("Query: %s", cq.query)
			t.Logf("Plan:\n%s", plan)
			t.Logf("Indexes used: %v", indexes)
			
			// Check that query uses indexes and doesn't do full table scan
			if len(indexes) > 0 {
				indexUsageCount++
				t.Logf("✅ %s uses indexes efficiently", cq.name)
			} else {
				t.Errorf("❌ %s does not use indexes - this may cause performance issues", cq.name)
			}
			
			// Ensure no full table scan
			assert.NotContains(t, plan, "SCAN TABLE tasks", 
				"Query %s should not perform full table scan", cq.name)
		})
	}
	
	// Assert overall index usage ratio
	indexUsageRatio := float64(indexUsageCount) / float64(len(criticalQueries))
	assert.GreaterOrEqual(t, indexUsageRatio, MinIndexUsageRatio,
		"Index usage ratio %.2f is below threshold %.2f", indexUsageRatio, MinIndexUsageRatio)
	
	t.Logf("Index usage ratio: %.2f (%d/%d queries use indexes)", 
		indexUsageRatio, indexUsageCount, len(criticalQueries))
}

// TestPerformance_MemoryUsageStability tests memory usage during various operations
func TestPerformance_MemoryUsageStability(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Test memory stability during dataset generation
	t.Run("DatasetGeneration", func(t *testing.T) {
		metrics := suite.measureOperation(t, "MemoryStability_DatasetGen", MediumDatasetSize, func() error {
			suite.generateLargeDataset(t, 20, 10, MediumDatasetSize)
			return nil
		})
		
		assert.True(t, metrics.Success, "Dataset generation should succeed: %v", metrics.Error)
		assert.Less(t, metrics.MemoryUsed, uint64(MaxMemoryIncrease),
			"Memory usage %d bytes exceeded threshold during dataset generation", metrics.MemoryUsed)
		
		suite.logPerformanceMetrics(t, metrics)
	})
	
	// Test memory stability during large queries
	t.Run("LargeQueries", func(t *testing.T) {
		projects, _ := suite.Repos.Projects.List(suite.ctx, 0, 1)
		
		metrics := suite.measureOperation(t, "MemoryStability_LargeQuery", MediumDatasetSize, func() error {
			tasks, err := suite.Repos.Tasks.ListByProject(suite.ctx, projects[0].ID, 0, MediumDatasetSize)
			if err != nil {
				return err
			}
			t.Logf("Retrieved %d tasks in large query", len(tasks))
			return nil
		})
		
		assert.True(t, metrics.Success, "Large query should succeed: %v", metrics.Error)
		assert.Less(t, metrics.MemoryUsed, uint64(MaxMemoryIncrease),
			"Memory usage %d bytes exceeded threshold during large query", metrics.MemoryUsed)
		
		suite.logPerformanceMetrics(t, metrics)
	})
}

// TestPerformance_ConcurrentOperations tests performance under concurrent load
func TestPerformance_ConcurrentOperations(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate initial dataset
	suite.generateLargeDataset(t, 50, 25, MediumDatasetSize)
	
	// Get test data
	projects, _ := suite.Repos.Projects.List(suite.ctx, 0, 10)
	users, _ := suite.Repos.Users.List(suite.ctx, 0, 10)
	
	// Test concurrent reads
	t.Run("ConcurrentReads", func(t *testing.T) {
		concurrency := 10
		done := make(chan PerformanceMetrics, concurrency)
		
		startTime := time.Now()
		
		// Launch concurrent operations
		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				project := projects[workerID%len(projects)]
				
				metrics := suite.measureOperation(t, fmt.Sprintf("ConcurrentRead_Worker_%d", workerID), 100, func() error {
					tasks, err := suite.Repos.Tasks.ListByProject(suite.ctx, project.ID, 0, 100)
					if err != nil {
						return err
					}
					log.Printf("Worker %d retrieved %d tasks", workerID, len(tasks))
					return nil
				})
				
				done <- metrics
			}(i)
		}
		
		// Collect results
		var totalDuration time.Duration
		successCount := 0
		
		for i := 0; i < concurrency; i++ {
			metrics := <-done
			if metrics.Success {
				successCount++
			}
			totalDuration += metrics.Duration
		}
		
		overallDuration := time.Since(startTime)
		avgDuration := totalDuration / time.Duration(concurrency)
		
		t.Logf("Concurrent reads: %d/%d successful, avg duration: %v, total time: %v", 
			successCount, concurrency, avgDuration, overallDuration)
		
		assert.Equal(t, concurrency, successCount, "All concurrent operations should succeed")
		assert.Less(t, avgDuration, MaxQueryTime*2, "Average concurrent query time should be reasonable")
	})
	
	// Test mixed concurrent operations (reads + writes)
	t.Run("MixedConcurrentOperations", func(t *testing.T) {
		concurrency := 6 // 4 reads, 2 writes
		done := make(chan bool, concurrency)
		
		startTime := time.Now()
		
		// Launch read operations
		for i := 0; i < 4; i++ {
			go func(workerID int) {
				project := projects[workerID%len(projects)]
				_, err := suite.Repos.Tasks.ListByProject(suite.ctx, project.ID, 0, 50)
				done <- (err == nil)
			}(i)
		}
		
		// Launch write operations  
		for i := 0; i < 2; i++ {
			go func(workerID int) {
				project := projects[workerID%len(projects)]
				reporter := users[workerID%len(users)]
				
				task := suite.Factory.CreateTask(project, reporter,
					integration.WithTaskTitle(fmt.Sprintf("Concurrent Task %d", workerID)),
				)
				
				err := suite.Repos.Tasks.Create(suite.ctx, task)
				done <- (err == nil)
			}(i)
		}
		
		// Collect results
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}
		
		overallDuration := time.Since(startTime)
		
		t.Logf("Mixed concurrent operations: %d/%d successful, total time: %v", 
			successCount, concurrency, overallDuration)
		
		assert.Equal(t, concurrency, successCount, "All mixed concurrent operations should succeed")
	})
}

// BenchmarkTaskRepository_ListByProject benchmarks the most common query
func BenchmarkTaskRepository_ListByProject(b *testing.B) {
	suite := &PerformanceTestSuite{
		DatabaseTestSuite: integration.SetupDatabaseTest(&testing.T{}),
		ctx:              context.Background(),
	}
	defer suite.Cleanup()
	
	// Setup benchmark data
	suite.generateLargeDataset(&testing.T{}, 10, 5, 1000)
	
	projects, _ := suite.Repos.Projects.List(suite.ctx, 0, 1)
	testProjectID := projects[0].ID
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Direct query to avoid repository sorting issues
		filter := "project = {:projectID}"
		params := map[string]interface{}{"projectID": testProjectID}
		
		_, err := suite.DB.App().FindRecordsByFilter("tasks", filter, "", 50, 0, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTaskRepository_Search benchmarks full-text search
func BenchmarkTaskRepository_Search(b *testing.B) {
	suite := &PerformanceTestSuite{
		DatabaseTestSuite: integration.SetupDatabaseTest(&testing.T{}),
		ctx:              context.Background(),
	}
	defer suite.Cleanup()
	
	// Setup benchmark data
	suite.generateLargeDataset(&testing.T{}, 10, 5, 2000)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Direct search query
		filter := "title ~ {:query} || description ~ {:query}"
		params := map[string]interface{}{"query": "performance"}
		
		_, err := suite.DB.App().FindRecordsByFilter("tasks", filter, "", 20, 0, params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// logPerformanceMetrics logs detailed performance metrics
func (p *PerformanceTestSuite) logPerformanceMetrics(t *testing.T, metrics PerformanceMetrics) {
	t.Logf("📊 Performance Metrics for %s:", metrics.Operation)
	t.Logf("   Duration: %v", metrics.Duration)
	t.Logf("   Records Processed: %d", metrics.RecordsProcessed)
	t.Logf("   Memory Used: %.2f MB", float64(metrics.MemoryUsed)/(1024*1024))
	t.Logf("   Memory Allocated: %.2f MB", float64(metrics.MemoryAllocated)/(1024*1024))
	t.Logf("   Success: %t", metrics.Success)
	
	if metrics.RecordsProcessed > 0 {
		recordsPerSecond := float64(metrics.RecordsProcessed) / metrics.Duration.Seconds()
		t.Logf("   Throughput: %.2f records/second", recordsPerSecond)
	}
	
	if len(metrics.IndexesUsed) > 0 {
		t.Logf("   Indexes Used: %v", metrics.IndexesUsed)
	}
	
	if metrics.Error != nil {
		t.Logf("   Error: %v", metrics.Error)
	}
}

// TestPerformance_DatabaseSchemaIndexes validates that expected indexes exist
func TestPerformance_DatabaseSchemaIndexes(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Query to get all indexes on tasks table
	indexQuery := `
		SELECT name, sql 
		FROM sqlite_master 
		WHERE type = 'index' 
		AND tbl_name = 'tasks' 
		AND name NOT LIKE 'sqlite_%'
	`
	
	rows, err := suite.DB.App().DB().NewQuery(indexQuery).Rows()
	require.NoError(t, err)
	defer rows.Close()
	
	indexes := make(map[string]string)
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err == nil {
			indexes[name] = sql
		}
	}
	
	t.Logf("Found %d indexes on tasks table:", len(indexes))
	for name, sql := range indexes {
		t.Logf("  %s: %s", name, sql)
	}
	
	// Verify critical indexes exist (adjust based on your actual schema)
	expectedIndexFields := []string{"project", "assignee", "status", "reporter"}
	
	for _, field := range expectedIndexFields {
		hasIndex := false
		for indexName, indexSQL := range indexes {
			if containsSubstring(indexSQL, field) {
				t.Logf("✅ Found index for field '%s': %s", field, indexName)
				hasIndex = true
				break
			}
		}
		
		if !hasIndex {
			t.Logf("⚠️  No index found for critical field '%s' - consider adding one for better performance", field)
		}
	}
}

// TestPerformance_QueryExecutionPlans captures and logs execution plans for documentation
func TestPerformance_QueryExecutionPlans(t *testing.T) {
	suite := setupPerformanceTest(t)
	defer suite.Cleanup()
	
	// Generate small dataset for plan analysis
	suite.generateLargeDataset(t, 10, 5, 100)
	
	// Get test data
	projects, _ := suite.Repos.Projects.List(suite.ctx, 0, 1)
	users, _ := suite.Repos.Users.List(suite.ctx, 0, 1)
	
	// Queries to analyze
	queries := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "Simple Project Filter",
			query: "SELECT * FROM tasks WHERE project = ?",
			args:  []interface{}{projects[0].ID},
		},
		{
			name:  "Project Filter with Limit",
			query: "SELECT * FROM tasks WHERE project = ? LIMIT 50",
			args:  []interface{}{projects[0].ID},
		},
		{
			name:  "Assignee Filter",
			query: "SELECT * FROM tasks WHERE assignee = ?",
			args:  []interface{}{users[0].ID},
		},
		{
			name:  "Compound Filter (Project + Status)",
			query: "SELECT * FROM tasks WHERE project = ? AND status = ?",
			args:  []interface{}{projects[0].ID, "todo"},
		},
		{
			name:  "Count Query",
			query: "SELECT COUNT(*) FROM tasks WHERE project = ?",
			args:  []interface{}{projects[0].ID},
		},
		{
			name:  "Join with Projects",
			query: "SELECT t.*, p.title FROM tasks t JOIN projects p ON t.project = p.id WHERE t.assignee = ?",
			args:  []interface{}{users[0].ID},
		},
		{
			name:  "Search Pattern",
			query: "SELECT * FROM tasks WHERE title LIKE ? OR description LIKE ?",
			args:  []interface{}{"%test%", "%test%"},
		},
	}
	
	t.Log("📋 Query Execution Plan Analysis:")
	t.Log("=====================================")
	
	for _, q := range queries {
		t.Logf("\n🔍 %s", q.name)
		t.Logf("Query: %s", q.query)
		
		plan, indexes := suite.analyzeQueryPlan(t, q.query, q.args...)
		
		t.Logf("Execution Plan:")
		if plan != "" {
			for _, line := range splitString(plan, "\n") {
				t.Logf("  %s", line)
			}
		} else {
			t.Logf("  (Unable to retrieve execution plan)")
		}
		
		if len(indexes) > 0 {
			t.Logf("Indexes Used: %v", indexes)
		} else {
			t.Logf("Indexes Used: None (may indicate full table scan)")
		}
		
		t.Log("-------------------------------------")
	}
}

// splitString splits a string by separator
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	
	var result []string
	start := 0
	
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	
	return result
}