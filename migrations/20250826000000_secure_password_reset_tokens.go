package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// For security reasons, we need to clear all existing password reset tokens
		// because we're changing from plaintext storage to HMAC-SHA256 hashes.
		// Existing plaintext tokens cannot be converted to hashes.
		
		collection, err := app.FindCollectionByNameOrId("password_reset_tokens")
		if err != nil {
			// Collection doesn't exist, nothing to clear
			return nil
		}

		// Delete all existing password reset tokens
		records, err := app.FindRecordsByFilter(
			collection.Id,
			"", // no filter, get all records
			"",
			0,
			0,
		)
		if err != nil {
			return err
		}

		for _, record := range records {
			if err := app.Delete(record); err != nil {
				// Continue deleting other records even if one fails
				continue
			}
		}

		return nil
	}, func(_ core.App) error {
		// Rollback: Nothing to rollback since we only deleted data
		// The original tokens can't be recovered anyway
		return nil
	})
}