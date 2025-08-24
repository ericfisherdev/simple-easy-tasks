// Package migrations contains PocketBase migrations for the task management system
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(_ core.App) error {
		// This migration will be handled through PocketBase admin UI
		// or through collection JSON definitions
		return nil
	}, func(_ core.App) error {
		// Rollback - remove collections through admin UI if needed
		return nil
	}, "20250824154017_create_tags_collection.go")
}
