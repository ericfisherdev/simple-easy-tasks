package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Add archived and archived_at fields to tasks collection

		collection, err := app.FindCollectionByNameOrId("tasks")
		if err != nil {
			return err
		}

		// Create archived field (boolean, default false)
		archivedField := &core.BoolField{
			Id:   "archived_field",
			Name: "archived",
		}

		// Create archived_at field (nullable datetime)
		archivedAtField := &core.DateField{
			Id:   "archived_at_field",
			Name: "archived_at",
			Min:  types.DateTime{},
			Max:  types.DateTime{},
		}

		// Add fields to the collection
		collection.Fields.Add(archivedField, archivedAtField)

		return app.Save(collection)
	}, func(app core.App) error {
		// Rollback: Remove the added fields

		collection, err := app.FindCollectionByNameOrId("tasks")
		if err != nil {
			return err
		}

		// Remove fields by name
		collection.Fields.RemoveByName("archived")
		collection.Fields.RemoveByName("archived_at")

		return app.Save(collection)
	})
}
