// Package migrations contains PocketBase migrations for the task management system
package migrations

import (
	_ "embed"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//go:embed collections.json
var collectionsSnapshot []byte

func init() {
	m.Register(func(app core.App) error {
		// deleteMissing=false keeps unmanaged collections; set true if you want exact snapshot.
		return app.ImportCollectionsByMarshaledJSON(collectionsSnapshot, false)
	}, nil)
}
