//go:build integration
// +build integration

// Package integration provides database integration testing infrastructure
package integration

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations" // Import PocketBase system migrations
)

// TestDatabase manages the lifecycle of a test database instance
type TestDatabase struct {
	app     core.App
	dbPath  string
	cleanup func()
}

// TestDatabaseConfig configures test database behavior
type TestDatabaseConfig struct {
	// UseInMemory determines whether to use in-memory or file-based database
	UseInMemory bool
	// LogQueries enables SQL query logging for debugging
	LogQueries bool
	// AutoMigrate runs migrations automatically on creation
	AutoMigrate bool
}

// DefaultTestConfig returns the default test configuration
func DefaultTestConfig() *TestDatabaseConfig {
	return &TestDatabaseConfig{
		UseInMemory: true,
		LogQueries:  false,
		AutoMigrate: true,
	}
}

var testCounter int64

// NewTestDatabase creates a new isolated test database instance
func NewTestDatabase(t *testing.T, config ...*TestDatabaseConfig) *TestDatabase {
	cfg := DefaultTestConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	var dbPath string
	var cleanup func()

	// Always use temporary directory approach like PocketBase's own tests
	tempDir := t.TempDir()

	if cfg.UseInMemory {
		// For in-memory databases, we still need a DataDir but the DB will be in-memory
		// Use a unique identifier for isolation
		counter := atomic.AddInt64(&testCounter, 1)
		dbPath = fmt.Sprintf("file:memdb_%s_%d?mode=memory&cache=shared", t.Name(), counter)
		cleanup = func() {
			// In-memory databases are automatically cleaned up when connections close
		}
	} else {
		// Use temporary file-based database
		dbPath = filepath.Join(tempDir, "data.db")
		cleanup = func() {
			// t.TempDir() automatically cleans up
		}
	}

	// Create PocketBase app with test database using proper v0.29.3 API
	appConfig := core.BaseAppConfig{
		DataDir:          tempDir,
		EncryptionEnv:    "pb_test_env",
		DataMaxOpenConns: core.DefaultDataMaxOpenConns,
		DataMaxIdleConns: core.DefaultDataMaxIdleConns,
		AuxMaxOpenConns:  core.DefaultAuxMaxOpenConns,
		AuxMaxIdleConns:  core.DefaultAuxMaxIdleConns,
		QueryTimeout:     core.DefaultQueryTimeout,
	}

	app := core.NewBaseApp(appConfig)

	// Bootstrap the app
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Failed to bootstrap PocketBase app: %v", err)
	}

	// Run all system migrations to set up PocketBase tables
	if err := app.RunAllMigrations(); err != nil {
		cleanup()
		t.Fatalf("Failed to run system migrations: %v", err)
	}

	// Apply test collections if auto-migrate is enabled
	if cfg.AutoMigrate {
		// For integration tests, use our simplified test collections
		if err := ApplyTestMigration(app); err != nil {
			cleanup()
			t.Fatalf("Failed to apply test migrations: %v", err)
		}
	}

	testDB := &TestDatabase{
		app:     app,
		dbPath:  dbPath,
		cleanup: cleanup,
	}

	// Register cleanup with test
	t.Cleanup(testDB.Cleanup)

	return testDB
}

// App returns the PocketBase application instance
func (db *TestDatabase) App() core.App {
	return db.app
}

// DBPath returns the database file path (for debugging purposes)
func (db *TestDatabase) DBPath() string {
	return db.dbPath
}

// Cleanup closes the database and cleans up resources
func (db *TestDatabase) Cleanup() {
	if db.cleanup != nil {
		db.cleanup()
	}
}

// Reset clears all data from the database while preserving the schema
func (db *TestDatabase) Reset() error {
	// Get all collection names
	collections, err := db.app.FindAllCollections()
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}

	// Clear data from each collection
	for _, collection := range collections {
		if collection.IsView() {
			continue // Skip views
		}

		sql := fmt.Sprintf("DELETE FROM %s", collection.Name)
		if _, err := db.app.DB().NewQuery(sql).Execute(); err != nil {
			return fmt.Errorf("failed to clear collection %s: %w", collection.Name, err)
		}
	}

	return nil
}
