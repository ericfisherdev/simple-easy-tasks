package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const githubCollectionsSnapshot = `[
  {
    "id": "github_integrations",
    "name": "github_integrations", 
    "type": "base",
    "system": false,
    "schema": [
      {
        "id": "project_id",
        "name": "project_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "user_id", 
        "name": "user_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "repo_owner",
        "name": "repo_owner", 
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "repo_name",
        "name": "repo_name",
        "type": "text", 
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "repo_id",
        "name": "repo_id",
        "type": "number",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "install_id",
        "name": "install_id", 
        "type": "number",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "access_token_encrypted",
        "name": "access_token_encrypted",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "refresh_token_encrypted",
        "name": "refresh_token_encrypted",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "token_type",
        "name": "token_type",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "expires_at",
        "name": "expires_at",
        "type": "date",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      },
      {
        "id": "key_version",
        "name": "key_version",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "access_token_deprecated",
        "name": "access_token_deprecated",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "settings",
        "name": "settings",
        "type": "json",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "maxSize": 0
        }
      }
    ],
    "indexes": [
      "CREATE INDEX idx_github_integrations_project_id ON github_integrations (project_id)",
      "CREATE INDEX idx_github_integrations_user_id ON github_integrations (user_id)",
      "CREATE INDEX idx_github_integrations_repo ON github_integrations (repo_owner, repo_name)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  },
  {
    "id": "github_oauth_states",
    "name": "github_oauth_states",
    "type": "base", 
    "system": false,
    "schema": [
      {
        "id": "state",
        "name": "state",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": true,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "user_id",
        "name": "user_id", 
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "project_id",
        "name": "project_id",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "expires_at",
        "name": "expires_at",
        "type": "date",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      }
    ],
    "indexes": [
      "CREATE UNIQUE INDEX idx_github_oauth_states_state ON github_oauth_states (state)",
      "CREATE INDEX idx_github_oauth_states_expires ON github_oauth_states (expires_at)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  },
  {
    "id": "github_issue_mappings",
    "name": "github_issue_mappings",
    "type": "base",
    "system": false,
    "schema": [
      {
        "id": "integration_id",
        "name": "integration_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "task_id",
        "name": "task_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "issue_number",
        "name": "issue_number",
        "type": "number",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "issue_id",
        "name": "issue_id",
        "type": "number",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "sync_direction",
        "name": "sync_direction",
        "type": "select",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "maxSelect": 1,
          "values": ["both", "to_github", "from_github"]
        }
      },
      {
        "id": "last_synced_at",
        "name": "last_synced_at",
        "type": "date",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      }
    ],
    "indexes": [
      "CREATE INDEX idx_github_issue_mappings_integration ON github_issue_mappings (integration_id)",
      "CREATE INDEX idx_github_issue_mappings_task ON github_issue_mappings (task_id)",
      "CREATE UNIQUE INDEX idx_github_issue_mappings_unique ON github_issue_mappings (integration_id, issue_number)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  },
  {
    "id": "github_commit_links",
    "name": "github_commit_links",
    "type": "base",
    "system": false,
    "schema": [
      {
        "id": "integration_id",
        "name": "integration_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "task_id",
        "name": "task_id", 
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "commit_sha",
        "name": "commit_sha",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "commit_message",
        "name": "commit_message",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "commit_url",
        "name": "commit_url",
        "type": "url",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "exceptDomains": null,
          "onlyDomains": null
        }
      },
      {
        "id": "author_login",
        "name": "author_login",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      }
    ],
    "indexes": [
      "CREATE INDEX idx_github_commit_links_integration ON github_commit_links (integration_id)",
      "CREATE INDEX idx_github_commit_links_task ON github_commit_links (task_id)",
      "CREATE UNIQUE INDEX idx_github_commit_links_unique ON github_commit_links (integration_id, commit_sha)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  },
  {
    "id": "github_pr_mappings",
    "name": "github_pr_mappings",
    "type": "base",
    "system": false,
    "schema": [
      {
        "id": "integration_id",
        "name": "integration_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "task_id",
        "name": "task_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "pr_number",
        "name": "pr_number",
        "type": "number",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "pr_id",
        "name": "pr_id",
        "type": "number",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "noDecimal": true
        }
      },
      {
        "id": "pr_status",
        "name": "pr_status",
        "type": "select",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "maxSelect": 1,
          "values": ["open", "closed", "merged"]
        }
      },
      {
        "id": "branch_name",
        "name": "branch_name",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "merged_at",
        "name": "merged_at",
        "type": "date",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      }
    ],
    "indexes": [
      "CREATE INDEX idx_github_pr_mappings_integration ON github_pr_mappings (integration_id)",
      "CREATE INDEX idx_github_pr_mappings_task ON github_pr_mappings (task_id)",
      "CREATE UNIQUE INDEX idx_github_pr_mappings_unique ON github_pr_mappings (integration_id, pr_number)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  },
  {
    "id": "github_webhook_events",
    "name": "github_webhook_events",
    "type": "base",
    "system": false,
    "schema": [
      {
        "id": "integration_id",
        "name": "integration_id",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "event_type",
        "name": "event_type",
        "type": "text",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "action",
        "name": "action",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "id": "payload",
        "name": "payload",
        "type": "json",
        "system": false,
        "required": true,
        "presentable": false,
        "unique": false,
        "options": {
          "maxSize": 0
        }
      },
      {
        "id": "processed_at",
        "name": "processed_at",
        "type": "date",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": "",
          "max": ""
        }
      },
      {
        "id": "processing_error",
        "name": "processing_error",
        "type": "text",
        "system": false,
        "required": false,
        "presentable": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      }
    ],
    "indexes": [
      "CREATE INDEX idx_github_webhook_events_integration ON github_webhook_events (integration_id)",
      "CREATE INDEX idx_github_webhook_events_type ON github_webhook_events (event_type)",
      "CREATE INDEX idx_github_webhook_events_processed ON github_webhook_events (processed_at)"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  }
]`

func init() {
	m.Register(func(app core.App) error {
		return app.ImportCollectionsByMarshaledJSON([]byte(githubCollectionsSnapshot), false)
	}, func(app core.App) error {
		// Rollback - delete the GitHub collections
		collections := []string{
			"github_webhook_events",
			"github_pr_mappings",
			"github_commit_links",
			"github_issue_mappings",
			"github_oauth_states",
			"github_integrations",
		}

		for _, name := range collections {
			if collection, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(collection); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
