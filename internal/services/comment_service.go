package services

import (
	"context"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
)

// CommentService defines the interface for comment-related business logic.
type CommentService interface {
	// CreateComment creates a new comment on a task
	CreateComment(ctx context.Context, req domain.CreateCommentRequest, userID string) (*domain.Comment, error)

	// GetComment gets a comment by ID
	GetComment(ctx context.Context, commentID string, userID string) (*domain.Comment, error)

	// UpdateComment updates a comment
	UpdateComment(
		ctx context.Context,
		commentID string,
		req domain.UpdateCommentRequest,
		userID string,
	) (*domain.Comment, error)

	// DeleteComment deletes a comment
	DeleteComment(ctx context.Context, commentID string, userID string) error

	// ListTaskComments lists comments for a task
	ListTaskComments(ctx context.Context, taskID string, userID string, offset, limit int) ([]*domain.Comment, error)

	// GetCommentThread gets a comment thread (comment and its replies)
	GetCommentThread(ctx context.Context, commentID string, userID string) ([]*domain.Comment, error)
}

// commentService implements CommentService interface.
type commentService struct {
	commentRepo repository.CommentRepository
	taskRepo    repository.TaskRepository
	userRepo    repository.UserRepository
}

// NewCommentService creates a new comment service.
func NewCommentService(
	commentRepo repository.CommentRepository,
	taskRepo repository.TaskRepository,
	userRepo repository.UserRepository,
) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		taskRepo:    taskRepo,
		userRepo:    userRepo,
	}
}

// CreateComment creates a new comment on a task.
func (s *commentService) CreateComment(
	ctx context.Context,
	req domain.CreateCommentRequest,
	userID string,
) (*domain.Comment, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check if task exists and user has access
	task, err := s.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// For now, assume we have access if task exists
	// In a full implementation, we'd check project access here
	_ = task

	// If this is a reply, check if parent comment exists
	if req.ParentID != "" {
		parentComment, err := s.commentRepo.GetByID(ctx, req.ParentID)
		if err != nil {
			return nil, domain.NewNotFoundError("PARENT_COMMENT_NOT_FOUND", "Parent comment not found")
		}

		// Ensure parent comment is on the same task
		if parentComment.TaskID != req.TaskID {
			return nil, domain.NewValidationError("INVALID_PARENT", "Parent comment must be on the same task", nil)
		}
	}

	// Create comment
	comment := &domain.Comment{
		TaskID:   req.TaskID,
		AuthorID: userID,
		Content:  req.Content,
		Type:     req.Type,
	}

	// Set parent comment ID if this is a reply
	if req.ParentID != "" {
		comment.ParentCommentID = &req.ParentID
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, domain.NewInternalError("COMMENT_CREATE_FAILED", "Failed to create comment", err)
	}

	return comment, nil
}

// GetComment gets a comment by ID.
func (s *commentService) GetComment(ctx context.Context, commentID string, _ string) (*domain.Comment, error) {
	if commentID == "" {
		return nil, domain.NewValidationError("INVALID_COMMENT_ID", "Comment ID cannot be empty", nil)
	}

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the task (and thus the comment)
	task, err := s.taskRepo.GetByID(ctx, comment.TaskID)
	if err != nil {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// For now, assume access if task exists
	// In full implementation, check project access
	_ = task

	return comment, nil
}

// UpdateComment updates a comment.
func (s *commentService) UpdateComment(
	ctx context.Context,
	commentID string,
	req domain.UpdateCommentRequest,
	userID string,
) (*domain.Comment, error) {
	if commentID == "" {
		return nil, domain.NewValidationError("INVALID_COMMENT_ID", "Comment ID cannot be empty", nil)
	}

	// Get existing comment
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Check if user is the author or has admin privileges
	if comment.AuthorID != userID {
		// In a full implementation, we'd check if user is project admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Role != domain.AdminRole {
			return nil, domain.NewAuthorizationError("ACCESS_DENIED", "You can only edit your own comments")
		}
	}

	// Apply updates
	if req.Content != nil {
		comment.Content = *req.Content
	}

	// Validate updated comment
	if err := comment.Validate(); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, domain.NewInternalError("COMMENT_UPDATE_FAILED", "Failed to update comment", err)
	}

	return comment, nil
}

// DeleteComment deletes a comment.
func (s *commentService) DeleteComment(ctx context.Context, commentID string, userID string) error {
	if commentID == "" {
		return domain.NewValidationError("INVALID_COMMENT_ID", "Comment ID cannot be empty", nil)
	}

	// Get existing comment
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	// Check if user is the author or has admin privileges
	if comment.AuthorID != userID {
		// In a full implementation, we'd check if user is project admin
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user.Role != domain.AdminRole {
			return domain.NewAuthorizationError("ACCESS_DENIED", "You can only delete your own comments")
		}
	}

	// Delete from repository
	if err := s.commentRepo.Delete(ctx, commentID); err != nil {
		return domain.NewInternalError("COMMENT_DELETE_FAILED", "Failed to delete comment", err)
	}

	return nil
}

// ListTaskComments lists comments for a task.
func (s *commentService) ListTaskComments(
	ctx context.Context,
	taskID string,
	_ string,
	offset, limit int,
) ([]*domain.Comment, error) {
	if taskID == "" {
		return nil, domain.NewValidationError("INVALID_TASK_ID", "Task ID cannot be empty", nil)
	}

	// Check if user has access to the task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// For now, assume access if task exists
	// In full implementation, check project access
	_ = task

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	comments, err := s.commentRepo.ListByTask(ctx, taskID, offset, limit)
	if err != nil {
		return nil, domain.NewInternalError("COMMENT_LIST_FAILED", "Failed to list comments", err)
	}

	return comments, nil
}

// GetCommentThread gets a comment thread (comment and its replies).
func (s *commentService) GetCommentThread(ctx context.Context, commentID string, _ string) ([]*domain.Comment, error) {
	if commentID == "" {
		return nil, domain.NewValidationError("INVALID_COMMENT_ID", "Comment ID cannot be empty", nil)
	}

	// Get the root comment
	rootComment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Check if user has access to the task
	task, err := s.taskRepo.GetByID(ctx, rootComment.TaskID)
	if err != nil {
		return nil, domain.NewNotFoundError("TASK_NOT_FOUND", "Task not found")
	}

	// For now, assume access if task exists
	_ = task

	// Get all replies to this comment
	replies, err := s.commentRepo.ListReplies(ctx, commentID)
	if err != nil {
		return nil, domain.NewInternalError("REPLIES_LIST_FAILED", "Failed to list replies", err)
	}

	// Combine root comment with replies
	thread := make([]*domain.Comment, 0, len(replies)+1)
	thread = append(thread, rootComment)
	thread = append(thread, replies...)

	return thread, nil
}
