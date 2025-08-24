// Package services provides access control and rules configuration for PocketBase.
package services

import (
	"log"
)

// SetupAccessRules configures collection access rules and API permissions.
// Currently exported but not used - kept for future implementation.
func (p *PocketBaseService) SetupAccessRules() error {
	log.Printf("Access rules setup - handled via Admin UI in v0.29.3")
	return nil
}

// SetupAPIPermissions configures API-level permissions and hooks.
// Currently exported but not used - kept for future implementation.
func (p *PocketBaseService) SetupAPIPermissions() {
	log.Printf("API permissions setup - handled via Admin UI in v0.29.3")
}

// Note: In PocketBase v0.29.3, collection access rules and API permissions
// are typically configured through the Admin UI or JavaScript migrations.
// The programmatic setup of these rules requires different APIs than
// what was available in earlier versions.
