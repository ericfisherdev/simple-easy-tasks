package main

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db core.App) error {
		// This migration will be handled through PocketBase admin UI
		// or through collection JSON definitions
		return nil
	}, func(db core.App) error {
		// Rollback - remove collections through admin UI if needed
		return nil
	}, "20250824154047_create_comments_collection.go")
}
