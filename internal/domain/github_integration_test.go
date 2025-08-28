package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubWebhookEvent_JSONPayload(t *testing.T) {
	t.Run("StoreAndRetrieveJSONPayload", func(t *testing.T) {
		// Create test payload
		testPayload := map[string]interface{}{
			"action": "opened",
			"repository": map[string]interface{}{
				"name":      "test-repo",
				"full_name": "user/test-repo",
			},
			"issue": map[string]interface{}{
				"id":     123,
				"number": 1,
				"title":  "Test Issue",
				"body":   "This is a test issue",
			},
		}

		// Marshal to JSON bytes
		payloadBytes, err := json.Marshal(testPayload)
		require.NoError(t, err)

		// Create webhook event
		webhookEvent := &GitHubWebhookEvent{
			ID:            "webhook-123",
			IntegrationID: "integration-456",
			EventType:     "issues",
			Action:        "opened",
			Payload:       payloadBytes,
		}

		// Verify event fields are set correctly
		assert.Equal(t, "webhook-123", webhookEvent.ID)
		assert.Equal(t, "integration-456", webhookEvent.IntegrationID)
		assert.Equal(t, "issues", webhookEvent.EventType)
		assert.Equal(t, "opened", webhookEvent.Action)

		// Verify we can unmarshal the payload back
		var retrievedPayload map[string]interface{}
		err = json.Unmarshal(webhookEvent.Payload, &retrievedPayload)
		require.NoError(t, err)

		// Verify payload content
		assert.Equal(t, "opened", retrievedPayload["action"])

		repo, ok := retrievedPayload["repository"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-repo", repo["name"])
		assert.Equal(t, "user/test-repo", repo["full_name"])

		issue, ok := retrievedPayload["issue"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(123), issue["id"]) // JSON numbers are float64
		assert.Equal(t, float64(1), issue["number"])
		assert.Equal(t, "Test Issue", issue["title"])
		assert.Equal(t, "This is a test issue", issue["body"])
	})

	t.Run("JSONPayloadMarshalUnmarshal", func(t *testing.T) {
		webhookEvent := &GitHubWebhookEvent{
			ID:            "webhook-789",
			IntegrationID: "integration-abc",
			EventType:     "pull_request",
			Action:        "closed",
			Payload:       json.RawMessage(`{"action": "closed", "number": 42, "merged": true}`),
		}

		// Marshal the entire webhook event to JSON
		eventBytes, err := json.Marshal(webhookEvent)
		require.NoError(t, err)

		// Unmarshal it back
		var retrievedEvent GitHubWebhookEvent
		err = json.Unmarshal(eventBytes, &retrievedEvent)
		require.NoError(t, err)

		// Verify fields
		assert.Equal(t, "webhook-789", retrievedEvent.ID)
		assert.Equal(t, "integration-abc", retrievedEvent.IntegrationID)
		assert.Equal(t, "pull_request", retrievedEvent.EventType)
		assert.Equal(t, "closed", retrievedEvent.Action)

		// Verify payload can be parsed
		var payload map[string]interface{}
		err = json.Unmarshal(retrievedEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, "closed", payload["action"])
		assert.Equal(t, float64(42), payload["number"])
		assert.Equal(t, true, payload["merged"])
	})

	t.Run("EmptyPayloadHandling", func(t *testing.T) {
		webhookEvent := &GitHubWebhookEvent{
			ID:            "webhook-empty",
			IntegrationID: "integration-def",
			EventType:     "ping",
			Action:        "",
			Payload:       json.RawMessage(`{}`),
		}

		// Verify event fields
		assert.Equal(t, "webhook-empty", webhookEvent.ID)
		assert.Equal(t, "integration-def", webhookEvent.IntegrationID)
		assert.Equal(t, "ping", webhookEvent.EventType)
		assert.Equal(t, "", webhookEvent.Action)

		// Verify empty payload can be handled
		var payload map[string]interface{}
		err := json.Unmarshal(webhookEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Empty(t, payload)
	})
}
