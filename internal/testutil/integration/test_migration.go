// Package integration contains test-specific migration for integration tests
package integration

import (
	_ "embed"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// ApplyTestMigration applies the test-specific collections to the app
func ApplyTestMigration(app core.App) error {
	// Debug: Check what internal tables exist
	fmt.Printf("DEBUG: Checking internal PocketBase tables:\n")
	rows, err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '_%'").Rows()
	if err == nil {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				fmt.Printf("Error closing rows: %v\n", closeErr)
			}
		}()
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err == nil {
				fmt.Printf("  - %s\n", tableName)
			}
		}
	}

	// Let's try the simple JSON import again but with debug info
	fmt.Printf("DEBUG: Attempting JSON import...\n")
	if err := simpleJSONImport(app); err != nil {
		return fmt.Errorf("failed to import collections: %w", err)
	}

	return nil
}

//go:embed test_collections_minimal.json
var testCollectionsSnapshot []byte

// simpleJSONImport tries the JSON import approach with more debugging
func simpleJSONImport(app core.App) error {
	fmt.Printf("DEBUG: JSON data size: %d bytes\n", len(testCollectionsSnapshot))

	// Try the import
	if err := app.ImportCollectionsByMarshaledJSON(testCollectionsSnapshot, false); err != nil {
		fmt.Printf("DEBUG: Import failed: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG: Import succeeded\n")

	// Check what was created
	collections, _ := app.FindAllCollections()
	fmt.Printf("DEBUG: Collections after import:\n")
	for _, col := range collections {
		fmt.Printf("  - %s (type: %s)\n", col.Name, col.Type)
	}

	// Check what the users collection ID is
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err == nil {
		fmt.Printf("DEBUG: Users collection ID: %s\n", usersCol.Id)
	}

	// Check all collection IDs
	fmt.Printf("DEBUG: All collection IDs:\n")
	for _, col := range collections {
		fmt.Printf("  - %s: %s\n", col.Name, col.Id)
	}

	return handleProjectsCollection(app)
}

// handleProjectsCollection handles projects collection debugging
func handleProjectsCollection(app core.App) error {
	// Try to find projects specifically
	projectsCol, err := app.FindCollectionByNameOrId("projects")
	if err != nil {
		fmt.Printf("DEBUG: Failed to find projects collection: %v\n", err)
		return err
	}

	fmt.Printf("DEBUG: Projects collection found: %s (ID: %s)\n", projectsCol.Name, projectsCol.Id)

	return debugTableStructure(app)
}

// debugTableStructure checks database table structure
func debugTableStructure(app core.App) error {
	// Check if the table has columns
	fmt.Printf("DEBUG: Checking projects table structure:\n")
	if err := checkTableInfo(app, "PRAGMA table_info(projects)"); err != nil {
		return err
	}

	// Let's also check what's in the _collections table
	fmt.Printf("DEBUG: Checking _collections table structure:\n")
	if err := checkTableInfo(app, "PRAGMA table_info(_collections)"); err != nil {
		return err
	}

	// Check the actual data
	fmt.Printf("DEBUG: Collections table data:\n")
	return checkCollectionsData(app)
}

// checkTableInfo executes table info query and prints results
func checkTableInfo(app core.App, query string) error {
	rows, err := app.DB().NewQuery(query).Rows()
	if err != nil {
		fmt.Printf("DEBUG: Failed to get table info: %v\n", err)
		return nil // Non-fatal error
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Error closing rows: %v\n", closeErr)
		}
	}()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}
		if scanErr := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); scanErr == nil {
			fmt.Printf("  %d: %s (%s)\n", cid, name, dataType)
		}
	}
	return nil
}

// checkCollectionsData queries and displays collections data
func checkCollectionsData(app core.App) error {
	rows, err := app.DB().NewQuery("SELECT * FROM _collections WHERE name = 'projects'").Rows()
	if err != nil {
		fmt.Printf("DEBUG: Failed to query _collections: %v\n", err)
		return nil // Non-fatal error
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("Error closing rows: %v\n", closeErr)
		}
	}()

	for rows.Next() {
		// Scan all columns as strings for debugging
		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if scanErr := rows.Scan(valuePtrs...); scanErr == nil {
			for i, col := range columns {
				fmt.Printf("  %s: %v\n", col, values[i])
			}
		}
	}
	return nil
}
