# PocketBase Database Cleanup Fix

## Problem

The `TestDatabase` struct's `Cleanup()` method was not properly shutting down the PocketBase `core.App` instance, leading to leaked database connections.

## Solution

Updated the `Cleanup()` method to call `app.ResetBootstrapState()` which properly:

1. Stops the cron ticker
2. Closes all database connections (data and auxiliary)
3. Cleans up PocketBase resources in the correct order

## Code Changes

```go
// OLD CODE (PROBLEMATIC)
func (db *TestDatabase) Cleanup() {
	if db.cleanup != nil {
		db.cleanup()
	}
}

// NEW CODE (FIXED)
func (db *TestDatabase) Cleanup() {
	// First call the custom cleanup function (file cleanup, etc.)
	if db.cleanup != nil {
		db.cleanup()
	}
	
	// Then properly shutdown the PocketBase app and close database connections
	if db.app != nil {
		// Use PocketBase's official shutdown method to properly close all database connections
		// This stops the cron ticker and closes all database connections (data + auxiliary)
		if err := db.app.ResetBootstrapState(); err != nil {
			// Log the error but don't fail the test - cleanup should be best effort
			// In a test environment, we want cleanup to proceed even if there are issues
			// Use a simple print since we may not have access to a logger
			println("Warning: Failed to reset PocketBase bootstrap state during cleanup:", err.Error())
		}
	}
}
```

## Why This Fix Works

1. **Official PocketBase Method**: `ResetBootstrapState()` is the official way to shut down a PocketBase app instance in v0.29.3
2. **Proper Resource Cleanup**: It handles all database connections (concurrent, nonconcurrent, aux databases)  
3. **Error Handling**: The fix includes proper error handling without failing tests
4. **Order of Operations**: Custom cleanup runs first, then PocketBase shutdown

## Verification

The fix has been tested with multiple integration test runs showing:
- No database connection leaks
- Proper resource cleanup
- Tests can run repeatedly without issues
- Both in-memory and file-based databases work correctly

## References

Based on PocketBase v0.29.3 source code analysis:
- `/core/base.go` - `ResetBootstrapState()` implementation
- `/tests/app.go` - How PocketBase's own test utilities handle shutdown
- `/pocketbase.go` - How the main app handles graceful shutdown