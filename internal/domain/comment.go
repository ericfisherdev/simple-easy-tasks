package domain

import (
	"strings"
	"time"
)

// CommentType represents the type of comment.
type CommentType string

const (
	// CommentTypeRegular is a regular comment.
	CommentTypeRegular CommentType = "regular"
	// CommentTypeSystemMessage is a system-generated message.
	CommentTypeSystemMessage CommentType = "system"
	// CommentTypeStatusUpdate is a status update comment.
	CommentTypeStatusUpdate CommentType = "status_update"
	// CommentTypeAssignmentChange is an assignment change comment.
	CommentTypeAssignmentChange CommentType = "assignment_change"
)

// Comment represents a user comment on a task
type Comment struct {
	// 8-byte aligned fields first
	ParentCommentID *string   `json:"parent_comment_id" db:"parent_comment"`
	CreatedAt       time.Time `json:"created_at" db:"created"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated"`

	// String and slice fields
	ID          string      `json:"id" db:"id"`
	Content     string      `json:"content" db:"content"`
	TaskID      string      `json:"task_id" db:"task"`
	AuthorID    string      `json:"author_id" db:"author"`
	Type        CommentType `json:"type" db:"type"`
	Attachments []string    `json:"attachments" db:"-"`

	// 1-byte aligned fields
	IsEdited bool `json:"is_edited" db:"is_edited"`
}

// NewComment creates a new comment with default values
func NewComment(content, taskID, authorID string) *Comment {
	now := time.Now()
	return &Comment{
		Content:   content,
		TaskID:    taskID,
		AuthorID:  authorID,
		Type:      CommentTypeRegular,
		IsEdited:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate performs comprehensive validation of the comment
func (c *Comment) Validate() error {
	if c.Content == "" {
		return NewValidationError("content", "Comment content is required", nil)
	}
	if len(c.Content) > 10000 {
		return NewValidationError("content", "Comment content must not exceed 10000 characters", nil)
	}
	if c.TaskID == "" {
		return NewValidationError("task_id", "Task ID is required", nil)
	}
	if c.AuthorID == "" {
		return NewValidationError("author_id", "Author ID is required", nil)
	}
	if c.ParentCommentID != nil && *c.ParentCommentID == c.ID {
		return NewConflictError("circular_reference",
			"Comment cannot be its own parent")
	}
	if len(c.Attachments) > 10 {
		return NewValidationError("attachments", "Maximum 10 attachments allowed per comment", nil)
	}
	return nil
}

// SetParentComment establishes a parent-child relationship for threading
func (c *Comment) SetParentComment(parentID string) error {
	// Reject empty parent IDs
	if strings.TrimSpace(parentID) == "" {
		return NewValidationError("parent_comment_id", "Parent comment ID cannot be empty", nil)
	}

	// Check for self-reference
	if parentID == c.ID {
		return NewConflictError("circular_reference",
			"Comment cannot be its own parent")
	}

	// Set the parent comment ID and update timestamp
	c.ParentCommentID = &parentID
	c.UpdatedAt = time.Now()
	return nil
}

// UpdateContent modifies the comment content and marks it as edited
func (c *Comment) UpdateContent(content string) error {
	if content == "" {
		return NewValidationError("content", "Comment content is required", nil)
	}
	if len(content) > 10000 {
		return NewValidationError("content", "Comment content must not exceed 10000 characters", nil)
	}
	c.Content = content
	c.IsEdited = true
	c.UpdatedAt = time.Now()
	return nil
}

// AddAttachment adds a file attachment to the comment
func (c *Comment) AddAttachment(attachmentID string) error {
	if len(c.Attachments) >= 10 {
		return NewValidationError("attachments", "Maximum 10 attachments allowed per comment", nil)
	}
	for _, existing := range c.Attachments {
		if existing == attachmentID {
			return NewConflictError("duplicate_attachment",
				"Attachment already exists on this comment")
		}
	}
	c.Attachments = append(c.Attachments, attachmentID)
	c.UpdatedAt = time.Now()
	return nil
}

// RemoveAttachment removes a file attachment from the comment
func (c *Comment) RemoveAttachment(attachmentID string) error {
	for i, existing := range c.Attachments {
		if existing == attachmentID {
			c.Attachments = append(c.Attachments[:i], c.Attachments[i+1:]...)
			c.UpdatedAt = time.Now()
			return nil
		}
	}
	return NewNotFoundError("attachment", "Attachment not found: "+attachmentID)
}

// IsReply checks if this comment is a reply to another comment
func (c *Comment) IsReply() bool {
	return c.ParentCommentID != nil
}

// CreateCommentRequest represents the data needed to create a new comment.
type CreateCommentRequest struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	TaskID   string                 `json:"task_id" binding:"required"`
	ParentID string                 `json:"parent_id,omitempty"`
	Content  string                 `json:"content" binding:"required,min=1"`
	Type     CommentType            `json:"type,omitempty"`
}

// Validate validates the create comment request.
func (r *CreateCommentRequest) Validate() error {
	if err := ValidateRequired("task_id", r.TaskID, "INVALID_TASK_ID", "Task ID is required"); err != nil {
		return err
	}

	if err := ValidateRequired("content", r.Content, "INVALID_CONTENT", "Comment content is required"); err != nil {
		return err
	}

	return nil
}

// UpdateCommentRequest represents the data that can be updated for a comment.
type UpdateCommentRequest struct {
	Content  *string                `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
