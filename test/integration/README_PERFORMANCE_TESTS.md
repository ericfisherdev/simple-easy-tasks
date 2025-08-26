# Performance Test Suite Documentation

## Overview

This comprehensive performance test suite provides detailed analysis of PocketBase database operations, focusing on query performance, index effectiveness, and memory usage monitoring for the Task Management System.

## Test Coverage

### 1. Query Performance Tests

#### Core Query Operations
- **TasksByProject**: Tests performance of project-based task retrieval (most common operation)
- **TasksByAssignee**: Tests performance of user-assigned task queries
- **Pagination**: Tests pagination performance at various offsets (0, 1K, 5K, 9K records)
- **Search**: Tests full-text search performance across title and description fields

#### Performance Thresholds
- **Standard queries**: < 100ms
- **Pagination queries**: < 50ms  
- **Search queries**: < 200ms
- **Bulk operations**: 1-5 seconds depending on size

### 2. Index Effectiveness Tests

#### Index Usage Validation
- **Query Plan Analysis**: Captures SQLite `EXPLAIN QUERY PLAN` output
- **Index Detection**: Identifies when queries use indexes vs full table scans
- **Compound Index Testing**: Tests effectiveness of multi-column indexes
- **Critical Query Coverage**: Ensures all high-frequency queries use indexes

#### Current Findings
- ⚠️ **No custom indexes present** - All queries perform full table scans
- ✅ **Primary key indexes work** - Join operations use automatic PK indexes
- 🔍 **Recommendations**: Add indexes on `project`, `assignee`, `status`, `reporter` fields

### 3. Bulk Operations Benchmarks

#### Large Dataset Operations
- **Bulk Creation**: 1,000 record inserts with performance monitoring
- **Bulk Updates**: Mass status/progress updates across records
- **Bulk Deletion**: Batch deletion with transaction safety
- **Memory Stability**: Ensures operations don't cause memory leaks

#### Performance Targets
- **Bulk inserts**: < 5 seconds for 1,000 records
- **Bulk updates**: < 3 seconds for 1,000 records  
- **Bulk deletes**: < 2 seconds for 1,000 records
- **Memory increase**: < 100MB during operations

### 4. Concurrency Testing

#### Multi-User Scenarios
- **Concurrent Reads**: Multiple users querying simultaneously
- **Mixed Operations**: Concurrent read/write operations
- **Lock Contention**: Tests database locking behavior under load

### 5. Memory Usage Monitoring

#### Memory Metrics
- **Baseline Capture**: Records initial memory state
- **Operation Tracking**: Monitors memory during each test
- **Leak Detection**: Identifies unbounded memory growth
- **Garbage Collection**: Ensures proper cleanup between tests

## Test Infrastructure

### Large Dataset Generation
The test suite generates realistic test data:
- **10,000+ tasks** across multiple projects
- **100+ users** with varied assignment patterns
- **50+ projects** with realistic distributions
- **Varied task properties**: Status, priority, assignments

### Realistic Data Distribution
- **Status Distribution**: 20% each of Backlog, Todo, Developing, Review, Complete
- **Priority Spread**: Even distribution across Critical, High, Medium, Low
- **Assignment Patterns**: ~33% of tasks assigned, rest unassigned
- **Search Content**: Tasks contain searchable terms for full-text testing

## Usage Examples

### Running All Performance Tests
```bash
go test -v -tags=integration -run=TestPerformance ./test/integration/
```

### Running Specific Test Categories
```bash
# Index effectiveness only
go test -v -tags=integration -run=TestPerformance_.*Index ./test/integration/

# Query performance only  
go test -v -tags=integration -run=TestPerformance_.*Queries ./test/integration/

# Memory monitoring
go test -v -tags=integration -run=TestPerformance_Memory ./test/integration/
```

### Running Benchmarks
```bash
go test -v -tags=integration -bench=BenchmarkTaskRepository ./test/integration/
```

## Performance Metrics Output

Each test provides detailed metrics:

```
📊 Performance Metrics for TasksByProject:
   Duration: 2.908388ms
   Records Processed: 10000
   Memory Used: 0.00 MB
   Memory Allocated: 0.80 MB
   Success: true
   Throughput: 3438330.79 records/second
   Indexes Used: []
```

## Query Plan Analysis

The suite captures and analyzes SQLite execution plans:

```
🔍 Simple Project Filter
Query: SELECT * FROM tasks WHERE project = ?
Execution Plan:
  3|0|SCAN tasks
Indexes Used: None (may indicate full table scan)
```

## Database Schema Analysis

Validates expected indexes exist:
```
Found 0 indexes on tasks table:
⚠️  No index found for critical field 'project'
⚠️  No index found for critical field 'assignee'  
⚠️  No index found for critical field 'status'
⚠️  No index found for critical field 'reporter'
```

## Recommendations for Optimization

### Immediate Actions
1. **Add Performance Indexes**:
   ```sql
   CREATE INDEX idx_tasks_project ON tasks(project);
   CREATE INDEX idx_tasks_assignee ON tasks(assignee);
   CREATE INDEX idx_tasks_status ON tasks(status);
   CREATE INDEX idx_tasks_reporter ON tasks(reporter);
   ```

2. **Compound Indexes for Common Queries**:
   ```sql
   CREATE INDEX idx_tasks_project_status ON tasks(project, status);
   CREATE INDEX idx_tasks_assignee_status ON tasks(assignee, status);
   ```

### Full-Text Search Optimization
Consider implementing FTS (Full-Text Search) indexes for title/description searches:
```sql
CREATE VIRTUAL TABLE tasks_fts USING fts5(title, description, content=tasks);
```

### Performance Monitoring
- **Regular Testing**: Run performance tests on CI/CD pipeline
- **Regression Detection**: Alert on performance degradation > 20%
- **Production Monitoring**: Track query times in production logs
- **Index Utilization**: Monitor SQLite query plans in development

## Integration with Development Workflow

### Pre-commit Testing
```bash
# Add to git hooks
go test -tags=integration -run=TestPerformance_.*Validation -timeout=30s ./test/integration/
```

### Continuous Integration
Include performance tests in CI with reasonable timeouts and dataset sizes for fast feedback.

### Production Deployment
Run full performance suite against staging environment before production deployments.

## Future Enhancements

### Additional Test Coverage
- **Stress Testing**: Test behavior under extreme load
- **Connection Pool Testing**: Multiple concurrent connections
- **Transaction Performance**: Test complex multi-table operations
- **Cache Effectiveness**: If caching layer is added

### Advanced Monitoring
- **Query Performance Profiling**: Detailed timing analysis
- **Lock Contention Monitoring**: Database lock analysis
- **I/O Performance**: Disk read/write monitoring
- **Network Latency**: Client-server response times

## Troubleshooting

### Common Issues
1. **Test Timeouts**: Increase timeout for large dataset tests
2. **Memory Issues**: Reduce dataset size on constrained environments  
3. **SQLite Locks**: Ensure proper connection handling
4. **Collection Schema**: Verify test collections match expectations

### Performance Debugging
1. **Enable Development Mode**: Use PocketBase `--dev` flag for SQL logging
2. **Analyze Query Plans**: Use `EXPLAIN QUERY PLAN` for slow queries
3. **Monitor System Resources**: Check CPU, memory, and disk usage
4. **Profile Go Code**: Use `go test -cpuprofile` for Go-level profiling