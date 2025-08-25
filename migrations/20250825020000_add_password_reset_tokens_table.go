package migrations

import (
	_ "embed"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//go:embed collections_password_reset_tokens.json
var passwordResetTokensCollectionsSnapshot []byte

func init() {
	m.Register(func(app core.App) error {
		// Import password_reset_tokens collection
		return app.ImportCollectionsByMarshaledJSON(passwordResetTokensCollectionsSnapshot, false)
	}, func(app core.App) error {
		// Rollback: delete the password_reset_tokens collection
		collection, err := app.FindCollectionByNameOrId("password_reset_tokens")
		if err != nil {
			return nil // Collection doesn't exist, nothing to rollback
		}
		return app.Delete(collection)
	})
}
