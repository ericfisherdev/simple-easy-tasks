// Package main provides the PocketBase entry point for database migrations and admin UI
package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	// Import migrations package to register all migrations via init()
	_ "simple-easy-tasks/migrations"
)

func main() {
	app := pocketbase.New()

	// Register the migrate command with automigrate enabled
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true, // Automatically run migrations on server start
	})

	// Start PocketBase
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
