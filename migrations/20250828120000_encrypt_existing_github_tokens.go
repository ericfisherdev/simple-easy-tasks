package migrations

import (
	"fmt"
	"log"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// This migration will encrypt existing plaintext tokens
		// Implementation placeholder - requires encryption service integration

		// Find records with deprecated plaintext tokens
		records, err := app.FindRecordsByFilter("github_integrations", "access_token_deprecated != ''", "", 1000, 0)
		if err != nil {
			log.Printf("Warning: Could not find records to migrate: %v", err)
			return nil // Don't fail migration for missing records
		}

		log.Printf("Found %d GitHub integration records to encrypt", len(records))

		for _, record := range records {
			plaintextToken := record.GetString("access_token_deprecated")
			if plaintextToken == "" {
				continue
			}

			// TODO: Implement actual encryption using your chosen method (age/NaCl)
			// For now, we'll just move the token and mark for manual encryption
			// encryptedToken, err := encryptionService.Encrypt(plaintextToken, keyVersion)
			// if err != nil {
			//     return fmt.Errorf("failed to encrypt token for record %s: %w", record.Id, err)
			// }

			// Placeholder: mark as needing encryption
			record.Set("access_token_encrypted", fmt.Sprintf("NEEDS_ENCRYPTION:%s", plaintextToken))
			record.Set("token_type", "bearer")
			record.Set("key_version", "v1")

			// Clear the deprecated field
			record.Set("access_token_deprecated", "")

			if err := app.Save(record); err != nil {
				log.Printf("Warning: Failed to migrate record %s: %v", record.Id, err)
				continue
			}
		}

		log.Printf("Migration completed. %d records marked for encryption.", len(records))
		log.Printf("WARNING: Manual encryption step required. Update application code to:")
		log.Printf("1. Implement encryption service with age/NaCl")
		log.Printf("2. Replace NEEDS_ENCRYPTION: prefixes with actual encrypted tokens")
		log.Printf("3. Update repository layer to decrypt tokens on load")

		return nil
	}, func(_ core.App) error {
		// Rollback: Cannot rollback encryption placeholders safely
		// Manual intervention required to restore tokens from backup
		log.Printf("WARNING: Cannot automatically rollback token encryption migration")
		log.Printf("Manual restore from backup required if rollback is needed")
		return nil
	})
}
