package domain

import (
	"time"
)

type Comment struct {
	ID              string    `json:"id" db:"id"`
	Content         string    `json:"content" db:"content"`
	TaskID          string    `json:"task_id" db:"task"`
	AuthorID        string    `json:"author_id" db:"author"`
	ParentCommentID *string   `json:"parent_comment_id" db:"parent_comment"`
	Attachments     []string  `json:"attachments" db:"-"`
	IsEdited        bool      `json:"is_edited" db:"is_edited"`
	CreatedAt       time.Time `json:"created_at" db:"created"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated"`
}

func NewComment(content, taskID, authorID string) *Comment {
	now := time.Now()
	return &Comment{
		Content:   content,
		TaskID:    taskID,
		AuthorID:  authorID,
		IsEdited:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

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

func (c *Comment) SetParentComment(parentID string) error {
	if parentID == c.ID {
		return NewConflictError("circular_reference",
			"Comment cannot be its own parent")
	}
	c.ParentCommentID = &parentID
	c.UpdatedAt = time.Now()
	return nil
}

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

func (c *Comment) RemoveAttachment(attachmentID string) error {
	newAttachments := make([]string, 0, len(c.Attachments))
	found := false
	for _, existing := range c.Attachments {
		if existing != attachmentID {
			newAttachments = append(newAttachments, existing)
		} else {
			found = true
		}
	}
	if !found {
		return NewNotFoundError("attachment", "Attachment not found: "+attachmentID)
	}
	c.Attachments = newAttachments
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Comment) IsReply() bool {
	return c.ParentCommentID != nil
}
