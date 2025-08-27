package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// CacheManager handles caching strategies for improved performance
type CacheManager interface {
	// Task caching
	CacheTask(ctx context.Context, task *domain.Task) error
	GetCachedTask(ctx context.Context, taskID string) (*domain.Task, error)
	InvalidateTask(ctx context.Context, taskID string) error
	
	// Board state caching
	CacheBoardState(ctx context.Context, projectID string, board *KanbanBoard) error
	GetCachedBoardState(ctx context.Context, projectID string) (*KanbanBoard, error)
	InvalidateBoardState(ctx context.Context, projectID string) error
	
	// Query result caching
	CacheQueryResult(ctx context.Context, key string, result interface{}, ttl time.Duration) error
	GetCachedQueryResult(ctx context.Context, key string, dest interface{}) error
	InvalidateQueryPattern(ctx context.Context, pattern string) error
	
	// Statistics caching
	CacheStatistics(ctx context.Context, projectID string, stats *BoardStatistics) error
	GetCachedStatistics(ctx context.Context, projectID string) (*BoardStatistics, error)
	InvalidateStatistics(ctx context.Context, projectID string) error
	
	// User-specific caching
	CacheUserTasks(ctx context.Context, userID string, tasks []*domain.Task) error
	GetCachedUserTasks(ctx context.Context, userID string) ([]*domain.Task, error)
	InvalidateUserCache(ctx context.Context, userID string) error
	
	// Cache management
	FlushCache(ctx context.Context) error
	GetCacheStats(ctx context.Context) (*CacheStats, error)
}

// CacheBackend defines the interface for cache storage backends
type CacheBackend interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) bool
	Flush(ctx context.Context) error
	Stats(ctx context.Context) (*BackendStats, error)
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits           int64         `json:"hits"`
	Misses         int64         `json:"misses"`
	HitRatio       float64       `json:"hit_ratio"`
	Keys           int64         `json:"keys"`
	Memory         int64         `json:"memory_bytes"`
	Evictions      int64         `json:"evictions"`
	LastFlush      *time.Time    `json:"last_flush,omitempty"`
	AverageLatency time.Duration `json:"average_latency"`
}

// BackendStats provides backend-specific statistics
type BackendStats struct {
	Connected bool                   `json:"connected"`
	Keys      int64                  `json:"keys"`
	Memory    int64                  `json:"memory_bytes"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CacheConfig configures cache behavior
type CacheConfig struct {
	TaskTTL       time.Duration `json:"task_ttl"`
	BoardTTL      time.Duration `json:"board_ttl"`
	StatsTTL      time.Duration `json:"stats_ttl"`
	QueryTTL      time.Duration `json:"query_ttl"`
	UserTasksTTL  time.Duration `json:"user_tasks_ttl"`
	MaxMemory     int64         `json:"max_memory_bytes"`
	Partitioning  bool          `json:"partitioning"`
}

// cacheManager implements sophisticated caching strategies
type cacheManager struct {
	backend CacheBackend
	config  CacheConfig
	stats   *cacheStats
}

// cacheStats tracks cache performance internally
type cacheStats struct {
	hits    int64
	misses  int64
	evictions int64
}

// Cache key constants and patterns
const (
	TaskCachePrefix       = "task:"
	BoardCachePrefix      = "board:"
	StatsCachePrefix      = "stats:"
	QueryCachePrefix      = "query:"
	UserTasksCachePrefix  = "user_tasks:"
	
	// Cache key patterns for invalidation
	TaskPattern       = "task:*"
	BoardPattern      = "board:*"
	StatsPattern      = "stats:*"
	UserPattern       = "user_tasks:*"
	ProjectPattern    = "project:%s:*" // Format with project ID
)

// NewCacheManager creates a new cache manager
func NewCacheManager(backend CacheBackend, config CacheConfig) CacheManager {
	return &cacheManager{
		backend: backend,
		config:  config,
		stats:   &cacheStats{},
	}
}

// Task caching methods

func (cm *cacheManager) CacheTask(ctx context.Context, task *domain.Task) error {
	if task == nil || task.ID == "" {
		return domain.NewValidationError("INVALID_TASK", "Task cannot be nil or empty", nil)
	}

	key := cm.buildTaskKey(task.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return domain.NewInternalError("CACHE_MARSHAL_ERROR", "Failed to marshal task for caching", err)
	}

	if err := cm.backend.Set(ctx, key, data, cm.config.TaskTTL); err != nil {
		slog.Warn("Failed to cache task", 
			"task_id", task.ID, 
			"error", err.Error(),
		)
		// Don't fail the operation if caching fails
		return nil
	}

	return nil
}

func (cm *cacheManager) GetCachedTask(ctx context.Context, taskID string) (*domain.Task, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	key := cm.buildTaskKey(taskID)
	data, err := cm.backend.Get(ctx, key)
	if err != nil {
		cm.stats.misses++
		return nil, nil // Cache miss, not an error
	}

	var task domain.Task
	if err := json.Unmarshal(data, &task); err != nil {
		// Invalid cached data, delete it
		cm.backend.Delete(ctx, key)
		cm.stats.misses++
		return nil, nil
	}

	cm.stats.hits++
	return &task, nil
}

func (cm *cacheManager) InvalidateTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	key := cm.buildTaskKey(taskID)
	return cm.backend.Delete(ctx, key)
}

// Board state caching methods

func (cm *cacheManager) CacheBoardState(ctx context.Context, projectID string, board *KanbanBoard) error {
	if projectID == "" || board == nil {
		return domain.NewValidationError("INVALID_BOARD", "Project ID and board cannot be empty", nil)
	}

	key := cm.buildBoardKey(projectID)
	data, err := json.Marshal(board)
	if err != nil {
		return domain.NewInternalError("CACHE_MARSHAL_ERROR", "Failed to marshal board for caching", err)
	}

	if err := cm.backend.Set(ctx, key, data, cm.config.BoardTTL); err != nil {
		slog.Warn("Failed to cache board state",
			"project_id", projectID,
			"error", err.Error(),
		)
		return nil
	}

	return nil
}

func (cm *cacheManager) GetCachedBoardState(ctx context.Context, projectID string) (*KanbanBoard, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	key := cm.buildBoardKey(projectID)
	data, err := cm.backend.Get(ctx, key)
	if err != nil {
		cm.stats.misses++
		return nil, nil
	}

	var board KanbanBoard
	if err := json.Unmarshal(data, &board); err != nil {
		cm.backend.Delete(ctx, key)
		cm.stats.misses++
		return nil, nil
	}

	cm.stats.hits++
	return &board, nil
}

func (cm *cacheManager) InvalidateBoardState(ctx context.Context, projectID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	key := cm.buildBoardKey(projectID)
	return cm.backend.Delete(ctx, key)
}

// Query result caching methods

func (cm *cacheManager) CacheQueryResult(ctx context.Context, key string, result interface{}, ttl time.Duration) error {
	if key == "" || result == nil {
		return domain.NewValidationError("INVALID_CACHE_PARAMS", "Key and result cannot be empty", nil)
	}

	cacheKey := cm.buildQueryKey(key)
	data, err := json.Marshal(result)
	if err != nil {
		return domain.NewInternalError("CACHE_MARSHAL_ERROR", "Failed to marshal query result", err)
	}

	if ttl == 0 {
		ttl = cm.config.QueryTTL
	}

	return cm.backend.Set(ctx, cacheKey, data, ttl)
}

func (cm *cacheManager) GetCachedQueryResult(ctx context.Context, key string, dest interface{}) error {
	if key == "" || dest == nil {
		return domain.NewValidationError("INVALID_CACHE_PARAMS", "Key and destination cannot be empty", nil)
	}

	cacheKey := cm.buildQueryKey(key)
	data, err := cm.backend.Get(ctx, cacheKey)
	if err != nil {
		cm.stats.misses++
		return nil // Cache miss
	}

	if err := json.Unmarshal(data, dest); err != nil {
		cm.backend.Delete(ctx, cacheKey)
		cm.stats.misses++
		return nil
	}

	cm.stats.hits++
	return nil
}

func (cm *cacheManager) InvalidateQueryPattern(ctx context.Context, pattern string) error {
	if pattern == "" {
		return domain.NewValidationError("INVALID_PATTERN", "Pattern cannot be empty", nil)
	}

	fullPattern := QueryCachePrefix + pattern
	return cm.backend.DeletePattern(ctx, fullPattern)
}

// Statistics caching methods

func (cm *cacheManager) CacheStatistics(ctx context.Context, projectID string, stats *BoardStatistics) error {
	if projectID == "" || stats == nil {
		return domain.NewValidationError("INVALID_STATS", "Project ID and statistics cannot be empty", nil)
	}

	key := cm.buildStatsKey(projectID)
	data, err := json.Marshal(stats)
	if err != nil {
		return domain.NewInternalError("CACHE_MARSHAL_ERROR", "Failed to marshal statistics", err)
	}

	return cm.backend.Set(ctx, key, data, cm.config.StatsTTL)
}

func (cm *cacheManager) GetCachedStatistics(ctx context.Context, projectID string) (*BoardStatistics, error) {
	if projectID == "" {
		return nil, domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	key := cm.buildStatsKey(projectID)
	data, err := cm.backend.Get(ctx, key)
	if err != nil {
		cm.stats.misses++
		return nil, nil
	}

	var stats BoardStatistics
	if err := json.Unmarshal(data, &stats); err != nil {
		cm.backend.Delete(ctx, key)
		cm.stats.misses++
		return nil, nil
	}

	cm.stats.hits++
	return &stats, nil
}

func (cm *cacheManager) InvalidateStatistics(ctx context.Context, projectID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	key := cm.buildStatsKey(projectID)
	return cm.backend.Delete(ctx, key)
}

// User-specific caching methods

func (cm *cacheManager) CacheUserTasks(ctx context.Context, userID string, tasks []*domain.Task) error {
	if userID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	key := cm.buildUserTasksKey(userID)
	data, err := json.Marshal(tasks)
	if err != nil {
		return domain.NewInternalError("CACHE_MARSHAL_ERROR", "Failed to marshal user tasks", err)
	}

	return cm.backend.Set(ctx, key, data, cm.config.UserTasksTTL)
}

func (cm *cacheManager) GetCachedUserTasks(ctx context.Context, userID string) ([]*domain.Task, error) {
	if userID == "" {
		return nil, domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	key := cm.buildUserTasksKey(userID)
	data, err := cm.backend.Get(ctx, key)
	if err != nil {
		cm.stats.misses++
		return nil, nil
	}

	var tasks []*domain.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		cm.backend.Delete(ctx, key)
		cm.stats.misses++
		return nil, nil
	}

	cm.stats.hits++
	return tasks, nil
}

func (cm *cacheManager) InvalidateUserCache(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.NewValidationError("INVALID_USER_ID", "User ID cannot be empty", nil)
	}

	key := cm.buildUserTasksKey(userID)
	return cm.backend.Delete(ctx, key)
}

// Cache management methods

func (cm *cacheManager) FlushCache(ctx context.Context) error {
	return cm.backend.Flush(ctx)
}

func (cm *cacheManager) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	backendStats, err := cm.backend.Stats(ctx)
	if err != nil {
		return nil, domain.NewInternalError("CACHE_STATS_ERROR", "Failed to get backend stats", err)
	}

	totalRequests := cm.stats.hits + cm.stats.misses
	hitRatio := 0.0
	if totalRequests > 0 {
		hitRatio = float64(cm.stats.hits) / float64(totalRequests)
	}

	return &CacheStats{
		Hits:      cm.stats.hits,
		Misses:    cm.stats.misses,
		HitRatio:  hitRatio,
		Keys:      backendStats.Keys,
		Memory:    backendStats.Memory,
		Evictions: cm.stats.evictions,
	}, nil
}

// Cache invalidation strategies

// InvalidateProjectCache invalidates all cache entries related to a project
func (cm *cacheManager) InvalidateProjectCache(ctx context.Context, projectID string) error {
	if projectID == "" {
		return domain.NewValidationError("INVALID_PROJECT_ID", "Project ID cannot be empty", nil)
	}

	// Invalidate board state
	if err := cm.InvalidateBoardState(ctx, projectID); err != nil {
		slog.Warn("Failed to invalidate board cache", "project_id", projectID, "error", err)
	}

	// Invalidate statistics
	if err := cm.InvalidateStatistics(ctx, projectID); err != nil {
		slog.Warn("Failed to invalidate stats cache", "project_id", projectID, "error", err)
	}

	// Invalidate project-specific query results
	pattern := fmt.Sprintf(ProjectPattern, projectID)
	if err := cm.backend.DeletePattern(ctx, pattern); err != nil {
		slog.Warn("Failed to invalidate project query cache", "project_id", projectID, "error", err)
	}

	return nil
}

// InvalidateTaskRelatedCache invalidates caches that might be affected by task changes
func (cm *cacheManager) InvalidateTaskRelatedCache(ctx context.Context, task *domain.Task) error {
	if task == nil {
		return nil
	}

	// Invalidate the task itself
	if err := cm.InvalidateTask(ctx, task.ID); err != nil {
		slog.Warn("Failed to invalidate task cache", "task_id", task.ID, "error", err)
	}

	// Invalidate board state for the project
	if err := cm.InvalidateBoardState(ctx, task.ProjectID); err != nil {
		slog.Warn("Failed to invalidate board cache", "project_id", task.ProjectID, "error", err)
	}

	// Invalidate statistics for the project
	if err := cm.InvalidateStatistics(ctx, task.ProjectID); err != nil {
		slog.Warn("Failed to invalidate stats cache", "project_id", task.ProjectID, "error", err)
	}

	// Invalidate user task cache for assignee and reporter
	if task.AssigneeID != nil {
		if err := cm.InvalidateUserCache(ctx, *task.AssigneeID); err != nil {
			slog.Warn("Failed to invalidate user cache", "user_id", *task.AssigneeID, "error", err)
		}
	}
	if err := cm.InvalidateUserCache(ctx, task.ReporterID); err != nil {
		slog.Warn("Failed to invalidate user cache", "user_id", task.ReporterID, "error", err)
	}

	return nil
}

// Helper methods for building cache keys

func (cm *cacheManager) buildTaskKey(taskID string) string {
	return TaskCachePrefix + taskID
}

func (cm *cacheManager) buildBoardKey(projectID string) string {
	return BoardCachePrefix + projectID
}

func (cm *cacheManager) buildStatsKey(projectID string) string {
	return StatsCachePrefix + projectID
}

func (cm *cacheManager) buildQueryKey(key string) string {
	return QueryCachePrefix + key
}

func (cm *cacheManager) buildUserTasksKey(userID string) string {
	return UserTasksCachePrefix + userID
}

// BuildQueryKey creates a standardized query cache key
func (cm *cacheManager) BuildQueryKey(operation string, params ...string) string {
	key := operation
	for _, param := range params {
		key += ":" + param
	}
	return key
}

// BuildProjectQueryKey creates a project-specific query key
func (cm *cacheManager) BuildProjectQueryKey(projectID string, operation string, params ...string) string {
	key := "project:" + projectID + ":" + operation
	for _, param := range params {
		key += ":" + param
	}
	return key
}

// BuildFiltersKey creates a cache key from task filters
func (cm *cacheManager) BuildFiltersKey(filters repository.TaskFilters) string {
	key := "filters"
	
	if len(filters.Status) > 0 {
		key += ":status"
		for _, s := range filters.Status {
			key += "-" + string(s)
		}
	}
	
	if len(filters.Priority) > 0 {
		key += ":priority"
		for _, p := range filters.Priority {
			key += "-" + string(p)
		}
	}
	
	if filters.AssigneeID != nil {
		key += ":assignee-" + *filters.AssigneeID
	}
	
	if filters.Search != "" {
		key += ":search-" + filters.Search
	}
	
	if filters.SortBy != "" {
		key += ":sort-" + filters.SortBy + "-" + filters.SortOrder
	}
	
	key += ":limit-" + strconv.Itoa(filters.Limit)
	key += ":offset-" + strconv.Itoa(filters.Offset)
	
	return key
}