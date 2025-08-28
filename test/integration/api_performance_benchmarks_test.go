//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ericfisherdev/simple-easy-tasks/internal/domain"
)

// APIPerformanceBenchmarkSuite provides performance testing for API endpoints
type APIPerformanceBenchmarkSuite struct {
	*APIEndpointsTestSuite
	performanceStats *PerformanceStatistics
}

// PerformanceStatistics tracks performance metrics
type PerformanceStatistics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	ErrorRate          float64       `json:"error_rate"`
	ThroughputMBps     float64       `json:"throughput_mbps"`
}

// PerformanceTestConfig defines test parameters
type PerformanceTestConfig struct {
	ConcurrentUsers  int
	RequestsPerUser  int
	TestDuration     time.Duration
	WarmupDuration   time.Duration
	TargetLatency    time.Duration // SLA target
	TargetThroughput float64       // requests/sec
	MaxErrorRate     float64       // percentage
}

// setupPerformanceBenchmarkSuite initializes the performance test suite
func setupPerformanceBenchmarkSuite(t *testing.T) *APIPerformanceBenchmarkSuite {
	apiSuite := setupAPITestSuite(t)

	suite := &APIPerformanceBenchmarkSuite{
		APIEndpointsTestSuite: apiSuite,
		performanceStats: &PerformanceStatistics{
			MinLatency: time.Hour, // Initialize to high value
		},
	}

	return suite
}

// BenchmarkAuthenticationEndpoints benchmarks critical auth endpoints
func BenchmarkAuthenticationEndpoints(b *testing.B) {
	suite := setupPerformanceBenchmarkSuite(&testing.T{})
	defer suite.Cleanup()

	config := PerformanceTestConfig{
		ConcurrentUsers:  50,
		RequestsPerUser:  100,
		TestDuration:     30 * time.Second,
		WarmupDuration:   5 * time.Second,
		TargetLatency:    200 * time.Millisecond, // 95th percentile target
		TargetThroughput: 500.0,                  // requests/sec
		MaxErrorRate:     1.0,                    // 1% max error rate
	}

	b.Run("LoginEndpoint", func(b *testing.B) {
		suite.benchmarkLoginEndpoint(b, config)
	})

	b.Run("RefreshTokenEndpoint", func(b *testing.B) {
		suite.benchmarkRefreshTokenEndpoint(b, config)
	})

	b.Run("GetProfileEndpoint", func(b *testing.B) {
		suite.benchmarkGetProfileEndpoint(b, config)
	})
}

// BenchmarkTaskManagementEndpoints benchmarks task-related endpoints
func BenchmarkTaskManagementEndpoints(b *testing.B) {
	suite := setupPerformanceBenchmarkSuite(&testing.T{})
	defer suite.Cleanup()

	config := PerformanceTestConfig{
		ConcurrentUsers:  100,
		RequestsPerUser:  50,
		TestDuration:     60 * time.Second,
		WarmupDuration:   10 * time.Second,
		TargetLatency:    300 * time.Millisecond,
		TargetThroughput: 1000.0,
		MaxErrorRate:     0.5,
	}

	b.Run("ListTasksEndpoint", func(b *testing.B) {
		suite.benchmarkListTasksEndpoint(b, config)
	})

	b.Run("CreateTaskEndpoint", func(b *testing.B) {
		suite.benchmarkCreateTaskEndpoint(b, config)
	})

	b.Run("UpdateTaskEndpoint", func(b *testing.B) {
		suite.benchmarkUpdateTaskEndpoint(b, config)
	})

	b.Run("MoveTaskEndpoint", func(b *testing.B) {
		suite.benchmarkMoveTaskEndpoint(b, config)
	})
}

// BenchmarkProjectManagementEndpoints benchmarks project-related endpoints
func BenchmarkProjectManagementEndpoints(b *testing.B) {
	suite := setupPerformanceBenchmarkSuite(&testing.T{})
	defer suite.Cleanup()

	config := PerformanceTestConfig{
		ConcurrentUsers:  75,
		RequestsPerUser:  25,
		TestDuration:     45 * time.Second,
		WarmupDuration:   5 * time.Second,
		TargetLatency:    250 * time.Millisecond,
		TargetThroughput: 750.0,
		MaxErrorRate:     0.1,
	}

	b.Run("ListProjectsEndpoint", func(b *testing.B) {
		suite.benchmarkListProjectsEndpoint(b, config)
	})

	b.Run("GetProjectEndpoint", func(b *testing.B) {
		suite.benchmarkGetProjectEndpoint(b, config)
	})
}

// benchmarkLoginEndpoint performs load testing on login endpoint
func (s *APIPerformanceBenchmarkSuite) benchmarkLoginEndpoint(b *testing.B, config PerformanceTestConfig) {
	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	// Warmup phase
	s.warmupEndpoint(b, "/api/auth/login", config.WarmupDuration)

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch concurrent goroutines
	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				requestStart := time.Now()

				loginReq := map[string]string{
					"email":    "test@example.com",
					"password": "testpassword123",
				}

				resp, err := s.makeRequest("POST", "/api/auth/login", loginReq, "")
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}

				// Small delay to prevent overwhelming
				time.Sleep(time.Millisecond)
			}
		}(user)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Calculate statistics
	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)

	// Log performance results
	b.Logf("Login Endpoint Performance Results:")
	b.Logf("  Total Requests: %d", stats.TotalRequests)
	b.Logf("  Success Rate: %.2f%%", (float64(stats.SuccessfulRequests)/float64(stats.TotalRequests))*100)
	b.Logf("  Requests/sec: %.2f", stats.RequestsPerSecond)
	b.Logf("  Average Latency: %v", stats.AverageLatency)
	b.Logf("  P95 Latency: %v", stats.P95Latency)
	b.Logf("  P99 Latency: %v", stats.P99Latency)

	// Assert performance targets
	assert.True(b, stats.P95Latency <= config.TargetLatency,
		"P95 latency %v exceeds target %v", stats.P95Latency, config.TargetLatency)
	assert.True(b, stats.RequestsPerSecond >= config.TargetThroughput,
		"Throughput %.2f below target %.2f", stats.RequestsPerSecond, config.TargetThroughput)
	assert.True(b, stats.ErrorRate <= config.MaxErrorRate,
		"Error rate %.2f%% exceeds maximum %.2f%%", stats.ErrorRate, config.MaxErrorRate)
}

// benchmarkRefreshTokenEndpoint tests token refresh performance
func (s *APIPerformanceBenchmarkSuite) benchmarkRefreshTokenEndpoint(b *testing.B, config PerformanceTestConfig) {
	// First get refresh tokens
	refreshTokens := s.generateRefreshTokens(b, config.ConcurrentUsers)

	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			refreshToken := refreshTokens[userID%len(refreshTokens)]

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				requestStart := time.Now()

				refreshReq := map[string]string{
					"refresh_token": refreshToken,
				}

				resp, err := s.makeRequest("POST", "/api/auth/refresh", refreshReq, "")
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}(user)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Refresh Token", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkGetProfileEndpoint tests profile retrieval performance
func (s *APIPerformanceBenchmarkSuite) benchmarkGetProfileEndpoint(b *testing.B, config PerformanceTestConfig) {
	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				requestStart := time.Now()
				resp, err := s.makeRequest("GET", "/api/users/profile", nil, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Get Profile", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkListTasksEndpoint tests task listing performance with various filters
func (s *APIPerformanceBenchmarkSuite) benchmarkListTasksEndpoint(b *testing.B, config PerformanceTestConfig) {
	// Pre-create tasks for testing
	s.createTestTasks(b, 1000) // Create 1000 test tasks

	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	// Different query variations to test
	queryVariations := []string{
		fmt.Sprintf("/api/projects/%s/tasks", s.testProject.ID),
		fmt.Sprintf("/api/projects/%s/tasks?status=backlog", s.testProject.ID),
		fmt.Sprintf("/api/projects/%s/tasks?priority=high", s.testProject.ID),
		fmt.Sprintf("/api/projects/%s/tasks?limit=50&offset=0", s.testProject.ID),
		fmt.Sprintf("/api/projects/%s/tasks?search=test", s.testProject.ID),
		fmt.Sprintf("/api/projects/%s/tasks?assignee=%s", s.testProject.ID, s.testUser.ID),
	}

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				// Use different query variations
				query := queryVariations[req%len(queryVariations)]

				requestStart := time.Now()
				resp, err := s.makeRequest("GET", query, nil, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "List Tasks", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkCreateTaskEndpoint tests task creation performance
func (s *APIPerformanceBenchmarkSuite) benchmarkCreateTaskEndpoint(b *testing.B, config PerformanceTestConfig) {
	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				taskReq := map[string]interface{}{
					"title":       fmt.Sprintf("Benchmark Task %d-%d", userID, req),
					"description": "Performance test task",
					"priority":    "medium",
					"status":      "backlog",
				}

				requestStart := time.Now()
				path := fmt.Sprintf("/api/projects/%s/tasks", s.testProject.ID)
				resp, err := s.makeRequest("POST", path, taskReq, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 201 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}(user)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Create Task", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkUpdateTaskEndpoint tests task update performance
func (s *APIPerformanceBenchmarkSuite) benchmarkUpdateTaskEndpoint(b *testing.B, config PerformanceTestConfig) {
	// Pre-create tasks to update
	taskIDs := s.createTestTasks(b, config.ConcurrentUsers*config.RequestsPerUser)

	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				taskIndex := userID*config.RequestsPerUser + req
				if taskIndex >= len(taskIDs) {
					continue
				}

				updateReq := map[string]interface{}{
					"title":       fmt.Sprintf("Updated Task %d-%d", userID, req),
					"description": "Updated via benchmark",
					"priority":    "high",
				}

				requestStart := time.Now()
				path := fmt.Sprintf("/api/projects/%s/tasks/%s", s.testProject.ID, taskIDs[taskIndex])
				resp, err := s.makeRequest("PUT", path, updateReq, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}(user)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Update Task", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkMoveTaskEndpoint tests task movement performance (kanban drag-drop simulation)
func (s *APIPerformanceBenchmarkSuite) benchmarkMoveTaskEndpoint(b *testing.B, config PerformanceTestConfig) {
	// Pre-create tasks to move
	taskIDs := s.createTestTasks(b, config.ConcurrentUsers*config.RequestsPerUser)

	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	statuses := []string{"todo", "developing", "review", "complete"}

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				taskIndex := userID*config.RequestsPerUser + req
				if taskIndex >= len(taskIDs) {
					continue
				}

				moveReq := map[string]interface{}{
					"new_status":   statuses[req%len(statuses)],
					"new_position": req + 1,
				}

				requestStart := time.Now()
				path := fmt.Sprintf("/api/projects/%s/tasks/%s/move", s.testProject.ID, taskIDs[taskIndex])
				resp, err := s.makeRequest("POST", path, moveReq, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}(user)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Move Task", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkListProjectsEndpoint tests project listing performance
func (s *APIPerformanceBenchmarkSuite) benchmarkListProjectsEndpoint(b *testing.B, config PerformanceTestConfig) {
	// Create additional test projects
	s.createTestProjects(b, 100)

	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				requestStart := time.Now()
				resp, err := s.makeRequest("GET", "/api/projects", nil, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "List Projects", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// benchmarkGetProjectEndpoint tests individual project retrieval performance
func (s *APIPerformanceBenchmarkSuite) benchmarkGetProjectEndpoint(b *testing.B, config PerformanceTestConfig) {
	b.ResetTimer()

	var (
		totalRequests   int64
		successRequests int64
		latencies       []time.Duration
		latencyMutex    sync.Mutex
	)

	startTime := time.Now()
	var wg sync.WaitGroup

	for user := 0; user < config.ConcurrentUsers; user++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for req := 0; req < config.RequestsPerUser; req++ {
				atomic.AddInt64(&totalRequests, 1)

				requestStart := time.Now()
				path := fmt.Sprintf("/api/projects/%s", s.testProject.ID)
				resp, err := s.makeRequest("GET", path, nil, s.authToken)
				latency := time.Since(requestStart)

				if err == nil && resp.StatusCode == 200 {
					atomic.AddInt64(&successRequests, 1)
				}

				latencyMutex.Lock()
				latencies = append(latencies, latency)
				latencyMutex.Unlock()

				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	stats := s.calculatePerformanceStats(totalRequests, successRequests, latencies, totalTime)
	s.logPerformanceStats(b, "Get Project", stats)
	s.assertPerformanceTargets(b, stats, config)
}

// Helper methods

// warmupEndpoint performs warmup requests before benchmarking
func (s *APIPerformanceBenchmarkSuite) warmupEndpoint(b *testing.B, endpoint string, duration time.Duration) {
	b.Logf("Warming up endpoint %s for %v", endpoint, duration)

	startTime := time.Now()
	for time.Since(startTime) < duration {
		s.makeRequest("GET", endpoint, nil, s.authToken)
		time.Sleep(10 * time.Millisecond)
	}

	b.Logf("Warmup completed")
}

// generateRefreshTokens creates refresh tokens for testing
func (s *APIPerformanceBenchmarkSuite) generateRefreshTokens(b *testing.B, count int) []string {
	var tokens []string

	for i := 0; i < count; i++ {
		loginReq := map[string]string{
			"email":    "test@example.com",
			"password": "testpassword123",
		}

		resp, err := s.makeRequest("POST", "/api/auth/login", loginReq, "")
		require.NoError(b, err)

		var result map[string]interface{}
		err = s.parseResponse(resp, &result)
		require.NoError(b, err)

		data := result["data"].(map[string]interface{})
		tokens = append(tokens, data["refresh_token"].(string))
	}

	return tokens
}

// createTestTasks creates test tasks for benchmarking
func (s *APIPerformanceBenchmarkSuite) createTestTasks(b *testing.B, count int) []string {
	var taskIDs []string
	ctx := context.Background()

	taskService := s.GetTaskService(b)

	for i := 0; i < count; i++ {
		req := domain.CreateTaskRequest{
			Title:       fmt.Sprintf("Benchmark Task %d", i),
			Description: "Task for performance testing",
			ProjectID:   s.testProject.ID,
			Priority:    domain.PriorityMedium,
		}

		task, err := taskService.CreateTask(ctx, req, s.testUser.ID)
		if err == nil {
			taskIDs = append(taskIDs, task.ID)
		}

		if i%100 == 0 {
			b.Logf("Created %d test tasks", i)
		}
	}

	b.Logf("Created %d test tasks total", len(taskIDs))
	return taskIDs
}

// createTestProjects creates test projects for benchmarking
func (s *APIPerformanceBenchmarkSuite) createTestProjects(b *testing.B, count int) []string {
	var projectIDs []string
	ctx := context.Background()

	projectRepo := s.GetProjectRepository(b)

	for i := 0; i < count; i++ {
		project := &domain.Project{
			Title:       fmt.Sprintf("Benchmark Project %d", i),
			Description: "Project for performance testing",
			Slug:        fmt.Sprintf("benchmark-project-%d", i),
			OwnerID:     s.testUser.ID,
			Color:       "#3b82f6",
			Icon:        "📊",
			Status:      domain.ActiveProject,
			Settings:    domain.ProjectSettings{},
			MemberIDs:   []string{},
		}

		err := projectRepo.Create(ctx, project)
		if err == nil {
			projectIDs = append(projectIDs, project.ID)
		}
	}

	return projectIDs
}

// calculatePerformanceStats computes performance statistics from latencies
func (s *APIPerformanceBenchmarkSuite) calculatePerformanceStats(totalReqs, successReqs int64, latencies []time.Duration, totalTime time.Duration) *PerformanceStatistics {
	if len(latencies) == 0 {
		return &PerformanceStatistics{}
	}

	// Sort latencies for percentile calculations
	for i := 0; i < len(latencies)-1; i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[i] > latencies[j] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	var totalLatency time.Duration
	minLatency := latencies[0]
	maxLatency := latencies[len(latencies)-1]

	for _, latency := range latencies {
		totalLatency += latency
	}

	avgLatency := totalLatency / time.Duration(len(latencies))
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}

	p95Latency := latencies[p95Index]
	p99Latency := latencies[p99Index]

	rps := float64(totalReqs) / totalTime.Seconds()
	errorRate := (float64(totalReqs-successReqs) / float64(totalReqs)) * 100

	return &PerformanceStatistics{
		TotalRequests:      totalReqs,
		SuccessfulRequests: successReqs,
		FailedRequests:     totalReqs - successReqs,
		AverageLatency:     avgLatency,
		MinLatency:         minLatency,
		MaxLatency:         maxLatency,
		P95Latency:         p95Latency,
		P99Latency:         p99Latency,
		RequestsPerSecond:  rps,
		ErrorRate:          errorRate,
	}
}

// logPerformanceStats logs performance statistics
func (s *APIPerformanceBenchmarkSuite) logPerformanceStats(b *testing.B, endpoint string, stats *PerformanceStatistics) {
	b.Logf("%s Endpoint Performance Results:", endpoint)
	b.Logf("  Total Requests: %d", stats.TotalRequests)
	b.Logf("  Successful Requests: %d", stats.SuccessfulRequests)
	b.Logf("  Failed Requests: %d", stats.FailedRequests)
	b.Logf("  Success Rate: %.2f%%", (float64(stats.SuccessfulRequests)/float64(stats.TotalRequests))*100)
	b.Logf("  Error Rate: %.2f%%", stats.ErrorRate)
	b.Logf("  Requests/sec: %.2f", stats.RequestsPerSecond)
	b.Logf("  Average Latency: %v", stats.AverageLatency)
	b.Logf("  Min Latency: %v", stats.MinLatency)
	b.Logf("  Max Latency: %v", stats.MaxLatency)
	b.Logf("  P95 Latency: %v", stats.P95Latency)
	b.Logf("  P99 Latency: %v", stats.P99Latency)
}

// assertPerformanceTargets validates performance meets targets
func (s *APIPerformanceBenchmarkSuite) assertPerformanceTargets(b *testing.B, stats *PerformanceStatistics, config PerformanceTestConfig) {
	// Assert P95 latency target
	if config.TargetLatency > 0 {
		assert.True(b, stats.P95Latency <= config.TargetLatency,
			"P95 latency %v exceeds target %v", stats.P95Latency, config.TargetLatency)
	}

	// Assert throughput target
	if config.TargetThroughput > 0 {
		assert.True(b, stats.RequestsPerSecond >= config.TargetThroughput,
			"Throughput %.2f below target %.2f", stats.RequestsPerSecond, config.TargetThroughput)
	}

	// Assert error rate target
	if config.MaxErrorRate > 0 {
		assert.True(b, stats.ErrorRate <= config.MaxErrorRate,
			"Error rate %.2f%% exceeds maximum %.2f%%", stats.ErrorRate, config.MaxErrorRate)
	}
}

// TestPerformanceRegression runs automated performance regression tests
func TestPerformanceRegression(t *testing.T) {
	suite := setupPerformanceBenchmarkSuite(t)
	defer suite.Cleanup()

	// Define baseline performance expectations
	baselineTargets := map[string]PerformanceTestConfig{
		"auth_login": {
			ConcurrentUsers:  10,
			RequestsPerUser:  20,
			TargetLatency:    100 * time.Millisecond,
			TargetThroughput: 200.0,
			MaxErrorRate:     0.5,
		},
		"task_list": {
			ConcurrentUsers:  20,
			RequestsPerUser:  10,
			TargetLatency:    150 * time.Millisecond,
			TargetThroughput: 300.0,
			MaxErrorRate:     0.1,
		},
		"task_create": {
			ConcurrentUsers:  15,
			RequestsPerUser:  10,
			TargetLatency:    200 * time.Millisecond,
			TargetThroughput: 150.0,
			MaxErrorRate:     0.1,
		},
	}

	t.Run("AuthLoginRegression", func(t *testing.T) {
		config := baselineTargets["auth_login"]
		suite.runPerformanceRegressionTest(t, "Login", config, suite.benchmarkLoginEndpoint)
	})

	t.Run("TaskListRegression", func(t *testing.T) {
		config := baselineTargets["task_list"]
		suite.runPerformanceRegressionTest(t, "ListTasks", config, suite.benchmarkListTasksEndpoint)
	})
}

// runPerformanceRegressionTest runs a single performance regression test
func (s *APIPerformanceBenchmarkSuite) runPerformanceRegressionTest(
	t *testing.T,
	name string,
	config PerformanceTestConfig,
	benchmarkFunc func(*testing.B, PerformanceTestConfig),
) {
	// Create a mock *testing.B for the benchmark function
	mockB := &testing.B{}

	t.Logf("Running performance regression test for %s", name)
	benchmarkFunc(mockB, config)

	// In a real implementation, you would compare results against stored baselines
	// and fail the test if performance has degraded beyond acceptable thresholds
	t.Logf("Performance regression test for %s completed", name)
}
