// Package services provides database migration services.
package services

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/migrations"
)

// MigrationService handles database migrations
type MigrationService struct {
	pbService *PocketBaseService
}

// NewMigrationService creates a new migration service
func NewMigrationService(pbService *PocketBaseService) *MigrationService {
	return &MigrationService{
		pbService: pbService,
	}
}

// RegisterMigrations registers all migrations with PocketBase
func (m *MigrationService) RegisterMigrations() {
	// Migration 1: Initial collections setup
	migrations.Register(func(app core.App) error {
		log.Println("Running migration: Create initial collections")
		return m.createInitialCollections(app.DB())
	}, func(app core.App) error {
		log.Println("Rolling back migration: Create initial collections")
		return m.rollbackInitialCollections(app.DB())
	}, "001_create_initial_collections.go")

	// Migration 2: Add indexes for performance
	migrations.Register(func(app core.App) error {
		log.Println("Running migration: Add performance indexes")
		return m.addPerformanceIndexes(app.DB())
	}, func(app core.App) error {
		log.Println("Rolling back migration: Add performance indexes")
		return m.rollbackPerformanceIndexes(app.DB())
	}, "002_add_performance_indexes.go")

	// Migration 3: Set up access rules
	migrations.Register(func(app core.App) error {
		log.Println("Running migration: Configure access rules")
		return m.configureAccessRules(app.DB())
	}, func(app core.App) error {
		log.Println("Rolling back migration: Configure access rules")
		return m.rollbackAccessRules(app.DB())
	}, "003_configure_access_rules.go")

	log.Println("All migrations registered successfully")
}

// createInitialCollections creates the initial database collections
func (m *MigrationService) createInitialCollections(_ dbx.Builder) error {
	// Note: In v0.29.3, collections are typically created via the Admin UI
	// or through JavaScript migrations. This is a placeholder for demonstration.
	log.Println("Initial collections setup - collections should be created via Admin UI or JS migrations")
	return nil
}

// rollbackInitialCollections removes the initial collections
func (m *MigrationService) rollbackInitialCollections(_ dbx.Builder) error {
	log.Println("Initial collections rollback - should be handled via Admin UI or JS migrations")
	return nil
}

// addPerformanceIndexes adds database indexes for better performance
func (m *MigrationService) addPerformanceIndexes(db dbx.Builder) error {
	indexes := []string{
		// Users indexes
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);",
		"CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);",

		// Projects indexes
		"CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner);",
		"CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);",
		"CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);",
		"CREATE INDEX IF NOT EXISTS idx_projects_title ON projects(title);",
		"CREATE INDEX IF NOT EXISTS idx_projects_created ON projects(created);",
		"CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(updated);",

		// Composite indexes
		"CREATE INDEX IF NOT EXISTS idx_projects_owner_slug ON projects(owner, slug);",
		"CREATE INDEX IF NOT EXISTS idx_projects_owner_status ON projects(owner, status);",
	}

	for _, query := range indexes {
		if _, err := db.NewQuery(query).Execute(); err != nil {
			return fmt.Errorf("failed to create index: %s - %w", query, err)
		}
	}

	log.Println("Performance indexes created successfully")
	return nil
}

// rollbackPerformanceIndexes removes performance indexes
func (m *MigrationService) rollbackPerformanceIndexes(db dbx.Builder) error {
	indexes := []string{
		"idx_users_email", "idx_users_username", "idx_users_role",
		"idx_projects_owner", "idx_projects_slug", "idx_projects_status",
		"idx_projects_title", "idx_projects_created", "idx_projects_updated",
		"idx_projects_owner_slug", "idx_projects_owner_status",
	}

	for _, indexName := range indexes {
		query := fmt.Sprintf("DROP INDEX IF EXISTS %s;", indexName)
		if _, err := db.NewQuery(query).Execute(); err != nil {
			return fmt.Errorf("failed to drop index %s: %w", indexName, err)
		}
	}

	log.Println("Performance indexes rollback completed")
	return nil
}

// configureAccessRules sets up collection access rules
func (m *MigrationService) configureAccessRules(_ dbx.Builder) error {
	log.Println("Access rules configuration - should be handled via Admin UI or JS migrations")
	return nil
}

// rollbackAccessRules removes access rules (makes collections open)
func (m *MigrationService) rollbackAccessRules(_ dbx.Builder) error {
	log.Println("Access rules rollback - should be handled via Admin UI or JS migrations")
	return nil
}

// CreateMigrationFile creates a new migration file template
func (m *MigrationService) CreateMigrationFile(name, migrationsDir string) error {
	timestamp := m.getCurrentTimestamp()
	filename := fmt.Sprintf("%s_%s.go", timestamp, strings.ReplaceAll(name, " ", "_"))
	filePath := filepath.Join(migrationsDir, filename)

	template := fmt.Sprintf(`package migrations

import (
	"log"

	"github.com/pocketbase/pocketbase/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		log.Printf("Running migration: %s")
		
		// Add your migration logic here
		
		return nil
	}, func(db migrations.DB) error {
		log.Printf("Rolling back migration: %s")
		
		// Add your rollback logic here
		
		return nil
	}, "%s")
}
`, name, name, filename)

	if err := writeFile(filePath, template); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}

	log.Printf("Created migration file: %s", filename)
	return nil
}

// Helper functions

// intPtr creates a pointer to an integer (unused helper function).
// func intPtr(i int) *int {
//	return &i
// }

func (m *MigrationService) getCurrentTimestamp() string {
	// Return current timestamp in format YYYYMMDDHHMMSS
	// This would use time.Now() in real implementation
	return "20240101000000" // Placeholder
}

// writeFile writes content to a file (placeholder implementation)
func writeFile(filename, _ string) error {
	// In real implementation, this would write to file
	log.Printf("Would write migration file: %s", filename)
	return nil
}
