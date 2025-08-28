package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubWebhookService_SignatureVerification_Security(t *testing.T) {
	testBody := []byte(`{"test": "payload"}`)

	t.Run("FailClosed_NoSecretConfigured_AllowUnsignedFalse", func(t *testing.T) {
		service := &GitHubWebhookService{
			secret:        "",    // No secret configured
			allowUnsigned: false, // Production mode - fail closed
		}

		// Should reject unsigned webhook when no secret is configured and allowUnsigned is false
		result := service.verifySignature("", testBody)
		assert.False(t, result, "Should reject unsigned webhook in production mode")
	})

	t.Run("DevMode_NoSecretConfigured_AllowUnsignedTrue", func(t *testing.T) {
		service := &GitHubWebhookService{
			secret:        "",   // No secret configured
			allowUnsigned: true, // Development mode - allow unsigned
		}

		// Should allow unsigned webhook when explicitly enabled
		result := service.verifySignature("", testBody)
		assert.True(t, result, "Should allow unsigned webhook in development mode when explicitly enabled")
	})

	t.Run("ValidSignature_WithSecret", func(t *testing.T) {
		secret := "test-secret"
		service := &GitHubWebhookService{
			secret:        secret,
			allowUnsigned: false, // Doesn't matter when secret is present
		}

		// Generate valid signature
		validSignature := "sha256=bcc0e8932e51a04f90891a518797ee2c4d04d4381853191e916d85cfbcc0939a"

		result := service.verifySignature(validSignature, testBody)
		assert.True(t, result, "Should accept valid signature")
	})

	t.Run("InvalidSignature_WithSecret", func(t *testing.T) {
		secret := "test-secret"
		service := &GitHubWebhookService{
			secret:        secret,
			allowUnsigned: false, // Doesn't matter when secret is present
		}

		// Invalid signature
		invalidSignature := "sha256=invalid-signature-hash"

		result := service.verifySignature(invalidSignature, testBody)
		assert.False(t, result, "Should reject invalid signature")
	})

	t.Run("MissingSignaturePrefix_WithSecret", func(t *testing.T) {
		secret := "test-secret"
		service := &GitHubWebhookService{
			secret:        secret,
			allowUnsigned: false,
		}

		// Missing sha256= prefix
		invalidSignature := "52b582138706ac0c597e80cfe7a7bf862ecaaea9e10b5d58b91a37c6e99827dd"

		result := service.verifySignature(invalidSignature, testBody)
		assert.False(t, result, "Should reject signature without sha256= prefix")
	})

	t.Run("EmptySignature_WithSecret", func(t *testing.T) {
		secret := "test-secret"
		service := &GitHubWebhookService{
			secret:        secret,
			allowUnsigned: false,
		}

		result := service.verifySignature("", testBody)
		assert.False(t, result, "Should reject empty signature when secret is configured")
	})
}

func TestGitHubWebhookService_ProductionSafety(t *testing.T) {
	t.Run("DefaultConfiguration_IsSecure", func(t *testing.T) {
		// Simulate production configuration (how it's configured in container registration)
		service := &GitHubWebhookService{
			secret:        "",    // Common misconfiguration
			allowUnsigned: false, // Production default
		}

		testBody := []byte(`{"malicious": "payload"}`)

		// Should fail closed and reject unsigned requests
		result := service.verifySignature("", testBody)
		assert.False(t, result, "Production configuration should fail closed and reject unsigned webhooks")

		// Should also reject requests with any signature when no secret is configured
		result = service.verifySignature("sha256=fake-signature", testBody)
		assert.False(t, result, "Production configuration should reject any signature when no secret is configured")
	})

	t.Run("SecretConfigured_RequiresValidSignature", func(t *testing.T) {
		service := &GitHubWebhookService{
			secret:        "production-secret",
			allowUnsigned: false, // Production default
		}

		testBody := []byte(`{"action": "opened"}`)

		// Should reject unsigned requests even with secret configured
		result := service.verifySignature("", testBody)
		assert.False(t, result, "Should reject unsigned requests when secret is configured")

		// Should reject invalid signatures
		result = service.verifySignature("sha256=invalid", testBody)
		assert.False(t, result, "Should reject invalid signatures")
	})
}
