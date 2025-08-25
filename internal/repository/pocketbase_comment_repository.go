package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"simple-easy-tasks/internal/domain"
)

const (
	taskFilterQuery = "task = {:taskID}"
)

type pocketbaseCommentRepository struct {
	app core.App
}

// NewPocketBaseCommentRepository creates a new PocketBase comment repository.
func NewPocketBaseCommentRepository(app core.App) CommentRepository {
	return &pocketbaseCommentRepository{app: app}
}

// Create creates a new comment in PocketBase.
func (r *pocketbaseCommentRepository) Create(_ context.Context, comment *domain.Comment) error {
	if err := comment.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	collection, err := r.app.FindCollectionByNameOrId("comments")
	if err != nil {
		return fmt.Errorf("failed to find comments collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("content", comment.Content)
	record.Set("task", comment.TaskID)
	record.Set("author", comment.AuthorID)
	record.Set("type", string(comment.Type))
	record.Set("is_edited", comment.IsEdited)
	record.Set("attachments", comment.Attachments)

	// Set optional parent comment
	if comment.ParentCommentID != nil && *comment.ParentCommentID != "" {
		record.Set("parent_comment", *comment.ParentCommentID)
	} else {
		record.Set("parent_comment", "")
	}

	// Set timestamps if provided
	if !comment.CreatedAt.IsZero() {
		record.Set("created", comment.CreatedAt)
	}
	if !comment.UpdatedAt.IsZero() {
		record.Set("updated", comment.UpdatedAt)
	}
	if comment.ID != "" {
		record.Id = comment.ID
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to save comment record: %w", err)
	}

	// Update the comment with persisted values
	comment.ID = record.Id
	if createdTime := record.GetDateTime("created"); !createdTime.IsZero() {
		comment.CreatedAt = createdTime.Time()
	}
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		comment.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// GetByID retrieves a comment by its ID.
func (r *pocketbaseCommentRepository) GetByID(_ context.Context, id string) (*domain.Comment, error) {
	if id == "" {
		return nil, fmt.Errorf("comment ID cannot be empty")
	}

	record, err := r.app.FindRecordById("comments", id)
	if err != nil {
		return nil, fmt.Errorf("failed to find comment by ID %s: %w", id, err)
	}

	return r.recordToComment(record)
}

// Update updates an existing comment.
func (r *pocketbaseCommentRepository) Update(_ context.Context, comment *domain.Comment) error {
	if err := comment.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if comment.ID == "" {
		return fmt.Errorf("comment ID cannot be empty for update")
	}

	record, err := r.app.FindRecordById("comments", comment.ID)
	if err != nil {
		return fmt.Errorf("failed to find comment for update: %w", err)
	}

	record.Set("content", comment.Content)
	record.Set("type", string(comment.Type))
	record.Set("is_edited", comment.IsEdited)
	record.Set("attachments", comment.Attachments)
	record.Set("updated", time.Now().UTC())

	// Update optional parent comment
	if comment.ParentCommentID != nil && *comment.ParentCommentID != "" {
		record.Set("parent_comment", *comment.ParentCommentID)
	} else {
		record.Set("parent_comment", "")
	}

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update comment record: %w", err)
	}

	// Update the comment with the persisted timestamps
	if updatedTime := record.GetDateTime("updated"); !updatedTime.IsZero() {
		comment.UpdatedAt = updatedTime.Time()
	}

	return nil
}

// Delete deletes a comment by ID.
func (r *pocketbaseCommentRepository) Delete(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("comment ID cannot be empty")
	}

	record, err := r.app.FindRecordById("comments", id)
	if err != nil {
		return fmt.Errorf("failed to find comment for deletion: %w", err)
	}

	if err := r.app.Delete(record); err != nil {
		return fmt.Errorf("failed to delete comment record: %w", err)
	}

	return nil
}

// ListByTask retrieves comments for a specific task.
func (r *pocketbaseCommentRepository) ListByTask(
	_ context.Context, taskID string, offset, limit int,
) ([]*domain.Comment, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	filter := taskFilterQuery
	params := dbx.Params{"taskID": taskID}

	records, err := r.app.FindRecordsByFilter(
		"comments", filter, "created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments by task %s: %w", taskID, err)
	}

	return r.recordsToComments(records)
}

// ListByAuthor retrieves comments by a specific author.
func (r *pocketbaseCommentRepository) ListByAuthor(
	_ context.Context, authorID string, offset, limit int,
) ([]*domain.Comment, error) {
	if authorID == "" {
		return nil, fmt.Errorf("author ID cannot be empty")
	}

	filter := "author = {:authorID}"
	params := dbx.Params{"authorID": authorID}

	records, err := r.app.FindRecordsByFilter(
		"comments", filter, "-created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments by author %s: %w", authorID, err)
	}

	return r.recordsToComments(records)
}

// ListReplies retrieves replies to a specific comment.
func (r *pocketbaseCommentRepository) ListReplies(_ context.Context, parentID string) ([]*domain.Comment, error) {
	if parentID == "" {
		return nil, fmt.Errorf("parent comment ID cannot be empty")
	}

	filter := "parent_comment = {:parentID}"
	params := dbx.Params{"parentID": parentID}

	records, err := r.app.FindRecordsByFilter(
		"comments", filter, "created", 0, 0, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list replies to comment %s: %w", parentID, err)
	}

	return r.recordsToComments(records)
}

// GetThread retrieves a comment thread.
func (r *pocketbaseCommentRepository) GetThread(_ context.Context, rootCommentID string) ([]*domain.Comment, error) {
	if rootCommentID == "" {
		return nil, fmt.Errorf("root comment ID cannot be empty")
	}

	// Get the root comment first
	rootComment, err := r.GetByID(context.Background(), rootCommentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root comment: %w", err)
	}

	// Get all replies
	replies, err := r.ListReplies(context.Background(), rootCommentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}

	// Combine root comment with replies
	thread := make([]*domain.Comment, 0, len(replies)+1)
	thread = append(thread, rootComment)
	thread = append(thread, replies...)

	return thread, nil
}

// Count returns the total number of comments.
func (r *pocketbaseCommentRepository) Count(_ context.Context) (int, error) {
	total, err := r.app.CountRecords("comments")
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}

	return int(total), nil
}

// CountByTask returns the number of comments for a task.
func (r *pocketbaseCommentRepository) CountByTask(_ context.Context, taskID string) (int, error) {
	if taskID == "" {
		return 0, fmt.Errorf("task ID cannot be empty")
	}

	filter := taskFilterQuery
	params := dbx.Params{"taskID": taskID}

	records, err := r.app.FindRecordsByFilter("comments", filter, "", 0, 0, params)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments by task %s: %w", taskID, err)
	}

	return len(records), nil
}

// CountByAuthor returns the number of comments by an author.
func (r *pocketbaseCommentRepository) CountByAuthor(_ context.Context, authorID string) (int, error) {
	if authorID == "" {
		return 0, fmt.Errorf("author ID cannot be empty")
	}

	filter := "author = {:authorID}"
	params := dbx.Params{"authorID": authorID}

	records, err := r.app.FindRecordsByFilter("comments", filter, "", 0, 0, params)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments by author %s: %w", authorID, err)
	}

	return len(records), nil
}

// ExistsByID checks if a comment exists by ID.
func (r *pocketbaseCommentRepository) ExistsByID(_ context.Context, id string) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("comment ID cannot be empty")
	}

	_, err := r.app.FindRecordById("comments", id)
	if err != nil {
		if err.Error() == sqlNoRowsError {
			return false, nil
		}
		return false, fmt.Errorf("failed to check comment existence by ID %s: %w", id, err)
	}

	return true, nil
}

// Search searches comments by content.
func (r *pocketbaseCommentRepository) Search(
	_ context.Context, query string, taskID string, offset, limit int,
) ([]*domain.Comment, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	filter, params := buildSearchFilter("content", query, taskFilterQuery, taskID, "taskID")

	records, err := r.app.FindRecordsByFilter(
		"comments", filter, "-created", limit, offset, params,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search comments with query '%s': %w", query, err)
	}

	return r.recordsToComments(records)
}

// BulkDelete deletes multiple comments.
func (r *pocketbaseCommentRepository) BulkDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil // Nothing to delete
	}

	// Validate all IDs first
	for i, id := range ids {
		if id == "" {
			return fmt.Errorf("comment ID %d is empty", i)
		}
	}

	// Process deletes individually
	for i, id := range ids {
		if err := r.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete comment %d (ID: %s): %w", i, id, err)
		}
	}

	return nil
}

// DeleteByTask deletes all comments for a task.
func (r *pocketbaseCommentRepository) DeleteByTask(_ context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	filter := taskFilterQuery
	params := dbx.Params{"taskID": taskID}

	records, err := r.app.FindRecordsByFilter("comments", filter, "", 0, 0, params)
	if err != nil {
		return fmt.Errorf("failed to find comments for task %s: %w", taskID, err)
	}

	for _, record := range records {
		if err := r.app.Delete(record); err != nil {
			return fmt.Errorf("failed to delete comment %s for task %s: %w", record.Id, taskID, err)
		}
	}

	return nil
}

// SoftDelete marks a comment as deleted without removing it.
func (r *pocketbaseCommentRepository) SoftDelete(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("comment ID cannot be empty")
	}

	return r.updateRecordWithStatusAndTimestamp(id, "comments", "deleted", true, "deleted_at")
}

// Restore restores a soft-deleted comment.
func (r *pocketbaseCommentRepository) Restore(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("comment ID cannot be empty")
	}

	return r.updateRecordWithStatusRestore(id, "comments", "deleted", "deleted_at")
}

// recordToComment converts a PocketBase record to a domain.Comment.
func (r *pocketbaseCommentRepository) recordToComment(record *core.Record) (*domain.Comment, error) {
	comment := &domain.Comment{
		ID:        record.Id,
		Content:   record.GetString("content"),
		TaskID:    record.GetString("task"),
		AuthorID:  record.GetString("author"),
		Type:      domain.CommentType(record.GetString("type")),
		IsEdited:  record.GetBool("is_edited"),
		CreatedAt: record.GetDateTime("created").Time(),
		UpdatedAt: record.GetDateTime("updated").Time(),
	}

	// Handle optional parent comment
	if parentComment := record.GetString("parent_comment"); parentComment != "" {
		comment.ParentCommentID = &parentComment
	}

	// Handle attachments array
	var attachments []string
	if err := record.UnmarshalJSONField("attachments", &attachments); err == nil && len(attachments) > 0 {
		comment.Attachments = attachments
	}

	return comment, nil
}

// recordsToComments converts PocketBase records to domain.Comment slice.
func (r *pocketbaseCommentRepository) recordsToComments(records []*core.Record) ([]*domain.Comment, error) {
	comments := make([]*domain.Comment, len(records))
	for i, record := range records {
		comment, err := r.recordToComment(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to comment (ID: %s): %w", record.Id, err)
		}
		comments[i] = comment
	}
	return comments, nil
}

// buildSearchFilter creates a search filter with optional parent filter
func buildSearchFilter(searchField, searchTerm, parentFilter, parentID, parentParam string) (string, dbx.Params) {
	// Sanitize search query for LIKE operations
	sanitizedTerm := "%" + strings.ReplaceAll(searchTerm, "%", "\\%") + "%"

	filter := searchField + " ~ {:searchTerm}"
	params := dbx.Params{"searchTerm": sanitizedTerm}

	// Add parent filter if provided
	if parentID != "" {
		filter += " && " + parentFilter
		params[parentParam] = parentID
	}

	return filter, params
}

// updateRecordWithStatusAndTimestamp updates a record with status fields and timestamp
func (r *pocketbaseCommentRepository) updateRecordWithStatusAndTimestamp(
	recordID, collection, statusField string, statusValue interface{}, timestampField string,
) error {
	record, err := r.app.FindRecordById(collection, recordID)
	if err != nil {
		return fmt.Errorf("failed to find record for update: %w", err)
	}

	record.Set(statusField, statusValue)
	record.Set(timestampField, time.Now().UTC())
	record.Set("updated", time.Now().UTC())

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// updateRecordWithStatusRestore updates a record by clearing status and timestamp fields
func (r *pocketbaseCommentRepository) updateRecordWithStatusRestore(
	recordID, collection, statusField, timestampField string,
) error {
	record, err := r.app.FindRecordById(collection, recordID)
	if err != nil {
		return fmt.Errorf("failed to find record for restoration: %w", err)
	}

	record.Set(statusField, false)
	record.Set(timestampField, nil)
	record.Set("updated", time.Now().UTC())

	if err := r.app.Save(record); err != nil {
		return fmt.Errorf("failed to restore record: %w", err)
	}

	return nil
}
