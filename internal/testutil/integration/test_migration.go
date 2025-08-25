// Package integration contains test-specific migration for integration tests
package integration

import (
	"github.com/pocketbase/pocketbase/core"
)

// ApplyTestMigration applies the test-specific collections to the app
func ApplyTestMigration(_ core.App) error {
	// For now, just use default auth collections - custom fields will be added later
	// The important thing is to have working integration tests with basic functionality
	return nil
}
