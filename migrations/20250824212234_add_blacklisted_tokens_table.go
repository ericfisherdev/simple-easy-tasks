package migrations

import (
	_ "embed"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//go:embed collections_blacklist.json
var blacklistCollectionsSnapshot []byte

func init() {
	m.Register(func(app core.App) error {
		// Import blacklisted_tokens collection
		return app.ImportCollectionsByMarshaledJSON(blacklistCollectionsSnapshot, false)
	}, nil)
}
