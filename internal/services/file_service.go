// Package services provides business logic and service implementations
package services

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// FileService handles file operations with PocketBase
type FileService struct {
	app *pocketbase.PocketBase
}

// NewFileService creates a new file service
func NewFileService(app *pocketbase.PocketBase) *FileService {
	return &FileService{
		app: app,
	}
}

// GetProtectedFileURL generates a signed URL for a protected file
// This URL includes a token that allows temporary access to the protected file
func (fs *FileService) GetProtectedFileURL(collectionName string, recordID string, filename string) (string, error) {
	// Get the collection
	collection, err := fs.app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return "", fmt.Errorf("collection not found: %w", err)
	}

	// Get the record
	record, err := fs.app.FindRecordById(collection.Id, recordID)
	if err != nil {
		return "", fmt.Errorf("record not found: %w", err)
	}

	// Generate file token for protected access
	// Token expires in 1 hour by default
	token, err := fs.generateFileToken(record, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate file token: %w", err)
	}

	// Build the URL with the token
	baseURL := fs.app.Settings().Meta.AppURL
	if baseURL == "" {
		baseURL = "http://localhost:8090" // Default for development
	}

	// Format: /api/files/{collection}/{recordId}/{filename}?token={token}
	url := fmt.Sprintf("%s/api/files/%s/%s/%s?token=%s",
		baseURL,
		collectionName,
		recordID,
		filename,
		token,
	)

	return url, nil
}

// generateFileToken creates a signed token for file access
func (fs *FileService) generateFileToken(record *core.Record, _ string) (string, error) {
	// Generate the token using PocketBase's built-in token generation
	// This ensures compatibility with PocketBase's file serving endpoints
	token, err := record.NewFileToken()
	if err != nil {
		return "", fmt.Errorf("failed to create file token: %w", err)
	}

	return token, nil
}

// ValidateFileAccess checks if a user has access to a file
func (fs *FileService) ValidateFileAccess(userID string, collectionName string, recordID string) (bool, error) {
	// Get the collection
	collection, err := fs.app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return false, fmt.Errorf("collection not found: %w", err)
	}

	// Get the record
	record, err := fs.app.FindRecordById(collection.Id, recordID)
	if err != nil {
		return false, fmt.Errorf("record not found: %w", err)
	}

	// Check if the user has access based on collection rules
	// For tasks: check if user is assigned or is project member
	// For comments: check if user can view the associated task
	switch collectionName {
	case "tasks":
		return fs.validateTaskFileAccess(userID, record)
	case "comments":
		return fs.validateCommentFileAccess(userID, record)
	default:
		// Default: owner-only access
		return record.GetString("user_id") == userID, nil
	}
}

// validateTaskFileAccess checks if user can access task attachments
func (fs *FileService) validateTaskFileAccess(userID string, taskRecord *core.Record) (bool, error) {
	// Check if user is the assignee
	if taskRecord.GetString("assignee") == userID {
		return true, nil
	}

	// Check if user is the creator
	if taskRecord.GetString("created_by") == userID {
		return true, nil
	}

	// Check if user is a project member
	projectID := taskRecord.GetString("project")
	if projectID != "" {
		projectCollection, _ := fs.app.FindCollectionByNameOrId("projects")
		if projectCollection != nil {
			project, err := fs.app.FindRecordById(projectCollection.Id, projectID)
			if err == nil {
				members := project.GetStringSlice("members")
				for _, member := range members {
					if member == userID {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// validateCommentFileAccess checks if user can access comment attachments
func (fs *FileService) validateCommentFileAccess(userID string, commentRecord *core.Record) (bool, error) {
	// Check if user is the comment author
	if commentRecord.GetString("user") == userID {
		return true, nil
	}

	// Check if user has access to the associated task
	taskID := commentRecord.GetString("task")
	if taskID != "" {
		taskCollection, _ := fs.app.FindCollectionByNameOrId("tasks")
		if taskCollection != nil {
			taskRecord, err := fs.app.FindRecordById(taskCollection.Id, taskID)
			if err == nil {
				return fs.validateTaskFileAccess(userID, taskRecord)
			}
		}
	}

	return false, nil
}

// FileAccessMiddleware creates middleware for validating file access
func (fs *FileService) FileAccessMiddleware() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Extract file request parameters
		collection := e.Request.PathValue("collection")
		recordID := e.Request.PathValue("recordId")

		// Get authenticated user
		authRecord := e.Auth
		if authRecord == nil {
			// No authentication, return nil to continue
			return nil
		}

		// Validate access
		hasAccess, err := fs.ValidateFileAccess(authRecord.Id, collection, recordID)
		if err != nil || !hasAccess {
			// User doesn't have access - return a standard error
			return fmt.Errorf("access denied to this file")
		}

		return nil
	}
}
