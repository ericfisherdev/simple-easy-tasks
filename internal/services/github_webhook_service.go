package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"

	"simple-easy-tasks/internal/domain"
)

const (
	syncDirectionToGitHub = "to_github"
)

// GitHubWebhookService handles GitHub webhook events
type GitHubWebhookService struct {
	secret           string
	integrationRepo  GitHubIntegrationRepository
	webhookEventRepo GitHubWebhookEventRepository
	githubService    *GitHubService
	taskService      TaskService
	eventHandlers    map[string]WebhookEventHandler
}

// GitHubWebhookEventRepository manages webhook event persistence
type GitHubWebhookEventRepository interface {
	Create(ctx context.Context, event *domain.GitHubWebhookEvent) error
	GetByID(ctx context.Context, id string) (*domain.GitHubWebhookEvent, error)
	MarkProcessed(ctx context.Context, id string, processedAt time.Time) error
	MarkError(ctx context.Context, id string, errorMsg string) error
	ListUnprocessed(ctx context.Context, limit int) ([]*domain.GitHubWebhookEvent, error)
	CleanupOld(ctx context.Context, olderThan time.Time) error
}

// WebhookEventHandler defines the interface for handling specific webhook events
type WebhookEventHandler interface {
	Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error
	EventType() string
}

// NewGitHubWebhookService creates a new webhook service
func NewGitHubWebhookService(
	secret string,
	integrationRepo GitHubIntegrationRepository,
	webhookEventRepo GitHubWebhookEventRepository,
	githubService *GitHubService,
	taskService TaskService,
) *GitHubWebhookService {
	service := &GitHubWebhookService{
		secret:           secret,
		integrationRepo:  integrationRepo,
		webhookEventRepo: webhookEventRepo,
		githubService:    githubService,
		taskService:      taskService,
		eventHandlers:    make(map[string]WebhookEventHandler),
	}

	// Register default event handlers
	service.RegisterHandler(&PushEventHandler{service: service})
	service.RegisterHandler(&PullRequestEventHandler{service: service})
	service.RegisterHandler(&IssuesEventHandler{service: service})
	service.RegisterHandler(&IssueCommentEventHandler{service: service})
	service.RegisterHandler(&PullRequestReviewEventHandler{service: service})

	return service
}

// RegisterHandler registers a webhook event handler
func (s *GitHubWebhookService) RegisterHandler(handler WebhookEventHandler) {
	s.eventHandlers[handler.EventType()] = handler
}

// HandleWebhook processes incoming webhook requests
func (s *GitHubWebhookService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Verify signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if !s.verifySignature(signature, body) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Get event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Get delivery ID for idempotency
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	// Parse repository information to find integration
	var repoPayload struct {
		Repository *github.Repository `json:"repository"`
	}

	if unmarshalErr := json.Unmarshal(body, &repoPayload); unmarshalErr != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if repoPayload.Repository == nil {
		http.Error(w, "Missing repository information", http.StatusBadRequest)
		return
	}

	// Find integration
	integration, err := s.integrationRepo.GetByRepoFullName(
		ctx,
		repoPayload.Repository.GetOwner().GetLogin(),
		repoPayload.Repository.GetName(),
	)
	if err != nil {
		// No integration found - this is normal for repos we don't track
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check if event type is enabled
	if !integration.IsWebhookEventEnabled(eventType) {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Store webhook event
	webhookEvent := &domain.GitHubWebhookEvent{
		ID:            deliveryID,
		IntegrationID: integration.ID,
		EventType:     eventType,
		Payload:       string(body),
		CreatedAt:     time.Now(),
	}

	if err := s.webhookEventRepo.Create(ctx, webhookEvent); err != nil {
		// Log error but don't fail the webhook
		fmt.Printf("Failed to store webhook event: %v\n", err)
	}

	// Process event asynchronously
	go s.processWebhookEvent(context.Background(), integration, webhookEvent)

	w.WriteHeader(http.StatusOK)
}

// processWebhookEvent processes a webhook event
func (s *GitHubWebhookService) processWebhookEvent(ctx context.Context, integration *domain.GitHubIntegration, event *domain.GitHubWebhookEvent) {
	// Find handler for event type
	handler, exists := s.eventHandlers[event.EventType]
	if !exists {
		if err := s.webhookEventRepo.MarkError(ctx, event.ID, fmt.Sprintf("No handler for event type: %s", event.EventType)); err != nil {
			// TODO: Add proper logging for database error
			// Continue processing despite marking error
			_ = err // Acknowledge error without action
		}
		return
	}

	// Parse payload based on event type
	payload, err := s.parsePayload(event.EventType, []byte(event.Payload))
	if err != nil {
		if markErr := s.webhookEventRepo.MarkError(ctx, event.ID, fmt.Sprintf("Failed to parse payload: %v", err)); markErr != nil {
			// TODO: Add proper logging for database error
			// Continue processing despite marking error
			_ = markErr // Acknowledge error without action
		}
		return
	}

	// Handle event
	if err := handler.Handle(ctx, integration, payload); err != nil {
		if markErr := s.webhookEventRepo.MarkError(ctx, event.ID, fmt.Sprintf("Handler error: %v", err)); markErr != nil {
			// TODO: Add proper logging for database error
			// Continue processing despite marking error
			_ = markErr // Acknowledge error without action
		}
		return
	}

	// Mark as processed
	if err := s.webhookEventRepo.MarkProcessed(ctx, event.ID, time.Now()); err != nil {
		// TODO: Add proper logging for database error
		// Processing was successful despite marking error
		_ = err // Acknowledge error without action
	}
}

// verifySignature verifies the webhook signature
func (s *GitHubWebhookService) verifySignature(signature string, body []byte) bool {
	if s.secret == "" {
		return true // Skip verification if no secret configured
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSignature := signature[7:] // Remove "sha256=" prefix

	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(body)
	actualSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(actualSignature))
}

// parsePayload parses webhook payload based on event type
func (s *GitHubWebhookService) parsePayload(eventType string, payload []byte) (interface{}, error) {
	switch eventType {
	case "push":
		var event github.PushEvent
		err := json.Unmarshal(payload, &event)
		return &event, err
	case "pull_request":
		var event github.PullRequestEvent
		err := json.Unmarshal(payload, &event)
		return &event, err
	case "issues":
		var event github.IssuesEvent
		err := json.Unmarshal(payload, &event)
		return &event, err
	case "issue_comment":
		var event github.IssueCommentEvent
		err := json.Unmarshal(payload, &event)
		return &event, err
	case "pull_request_review":
		var event github.PullRequestReviewEvent
		err := json.Unmarshal(payload, &event)
		return &event, err
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

// Event Handlers

// PushEventHandler handles push events
type PushEventHandler struct {
	service *GitHubWebhookService
}

// EventType returns the GitHub event type this handler processes
func (h *PushEventHandler) EventType() string { return "push" }

// Handle processes GitHub push events and links commits to tasks
func (h *PushEventHandler) Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error {
	event, ok := payload.(*github.PushEvent)
	if !ok {
		return fmt.Errorf("invalid payload type for push event")
	}

	// Process each commit
	for _, commit := range event.Commits {
		// Parse task references from commit message
		taskRefs := h.service.githubService.ParseTaskReferencesFromCommit(commit.GetMessage())

		for _, taskRef := range taskRefs {
			// Find task by reference
			// This would need to be implemented based on your task reference format
			taskID := h.extractTaskIDFromReference(taskRef)
			if taskID == "" {
				continue
			}

			// Link commit to task
			err := h.service.githubService.LinkCommitToTask(
				ctx,
				integration.ID,
				taskID,
				commit.GetSHA(),
				commit.GetMessage(),
				commit.GetURL(),
				commit.GetAuthor().GetLogin(),
			)
			if err != nil {
				fmt.Printf("Failed to link commit %s to task %s: %v\n", commit.GetSHA(), taskID, err)
			}
		}
	}

	return nil
}

func (h *PushEventHandler) extractTaskIDFromReference(ref string) string {
	// Implement task ID extraction logic based on your reference format
	// This is a simplified implementation
	ref = strings.ToLower(strings.TrimSpace(ref))
	if strings.HasPrefix(ref, "task-") {
		return strings.TrimPrefix(ref, "task-")
	}
	return ""
}

// PullRequestEventHandler handles pull request events
type PullRequestEventHandler struct {
	service *GitHubWebhookService
}

// EventType returns the GitHub event type this handler processes
func (h *PullRequestEventHandler) EventType() string { return "pull_request" }

// Handle processes GitHub pull request events and manages PR mappings
func (h *PullRequestEventHandler) Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error {
	event, ok := payload.(*github.PullRequestEvent)
	if !ok {
		return fmt.Errorf("invalid payload type for pull request event")
	}

	pr := event.GetPullRequest()
	action := event.GetAction()

	// Extract task references from PR title and body
	taskRefs := h.service.githubService.ParseTaskReferencesFromCommit(pr.GetTitle() + " " + pr.GetBody())

	for _, taskRef := range taskRefs {
		taskID := h.extractTaskIDFromReference(taskRef)
		if taskID == "" {
			continue
		}

		// Handle different PR actions
		switch action {
		case "opened", "reopened":
			// Create or update PR mapping
			err := h.createOrUpdatePRMapping(ctx, integration, pr, taskID)
			if err != nil {
				return err
			}
		case "closed":
			if pr.GetMerged() {
				// PR was merged - potentially complete the task
				err := h.handleMergedPR(ctx, integration, pr, taskID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (h *PullRequestEventHandler) extractTaskIDFromReference(ref string) string {
	// Same implementation as PushEventHandler
	ref = strings.ToLower(strings.TrimSpace(ref))
	if strings.HasPrefix(ref, "task-") {
		return strings.TrimPrefix(ref, "task-")
	}
	return ""
}

func (h *PullRequestEventHandler) createOrUpdatePRMapping(ctx context.Context, integration *domain.GitHubIntegration, pr *github.PullRequest, taskID string) error {
	// Check if mapping already exists
	existing, err := h.service.githubService.prMappingRepo.GetByPRNumber(ctx, integration.ID, pr.GetNumber())
	if err == nil && existing != nil {
		// Update existing mapping
		existing.PRStatus = pr.GetState()
		existing.UpdatedAt = time.Now()
		return h.service.githubService.prMappingRepo.Update(ctx, existing)
	}

	// Create new mapping
	mapping := &domain.GitHubPRMapping{
		ID:            generateID(),
		IntegrationID: integration.ID,
		TaskID:        taskID,
		PRNumber:      pr.GetNumber(),
		PRID:          pr.GetID(),
		PRStatus:      pr.GetState(),
		BranchName:    pr.GetHead().GetRef(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return h.service.githubService.prMappingRepo.Create(ctx, mapping)
}

func (h *PullRequestEventHandler) handleMergedPR(ctx context.Context, integration *domain.GitHubIntegration, pr *github.PullRequest, taskID string) error {
	// Update PR mapping with merge information
	mapping, err := h.service.githubService.prMappingRepo.GetByPRNumber(ctx, integration.ID, pr.GetNumber())
	if err != nil {
		return err
	}

	mergedAt := pr.GetMergedAt()
	mapping.MergedAt = &mergedAt.Time
	mapping.PRStatus = "merged"
	mapping.UpdatedAt = time.Now()

	if updateErr := h.service.githubService.prMappingRepo.Update(ctx, mapping); updateErr != nil {
		return updateErr
	}

	// Optionally move task to completed status
	// This depends on your workflow - you might want to make this configurable
	task, err := h.service.taskService.GetTask(ctx, taskID, "")
	if err != nil {
		return err // Task might not exist
	}

	if task.Status != "complete" {
		_, err = h.service.taskService.UpdateTaskStatus(ctx, taskID, "complete", "")
		return err
	}

	return nil
}

// IssuesEventHandler handles issue events
type IssuesEventHandler struct {
	service *GitHubWebhookService
}

// EventType returns the GitHub event type this handler processes
func (h *IssuesEventHandler) EventType() string { return "issues" }

// Handle processes GitHub issue events and manages issue-task mappings
func (h *IssuesEventHandler) Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error {
	event, ok := payload.(*github.IssuesEvent)
	if !ok {
		return fmt.Errorf("invalid payload type for issues event")
	}

	issue := event.GetIssue()
	action := event.GetAction()

	// Check if this issue is linked to a task
	mapping, err := h.service.githubService.issueMappingRepo.GetByIssueNumber(ctx, integration.ID, issue.GetNumber())
	if err != nil {
		return nil // No mapping found - that's okay
	}

	// Sync issue changes to task based on action
	switch action {
	case "closed":
		return h.handleClosedIssue(ctx, mapping)
	case "reopened":
		return h.handleReopenedIssue(ctx, mapping)
	case "edited":
		return h.handleEditedIssue(ctx, integration, issue, mapping)
	}

	return nil
}

func (h *IssuesEventHandler) handleClosedIssue(ctx context.Context, mapping *domain.GitHubIssueMapping) error {
	if mapping.SyncDirection == syncDirectionToGitHub {
		return nil // Don't sync back to task
	}

	// Mark task as complete
	_, err := h.service.taskService.UpdateTaskStatus(ctx, mapping.TaskID, "complete", "")
	return err
}

func (h *IssuesEventHandler) handleReopenedIssue(ctx context.Context, mapping *domain.GitHubIssueMapping) error {
	if mapping.SyncDirection == syncDirectionToGitHub {
		return nil // Don't sync back to task
	}

	// Reopen task
	task, err := h.service.taskService.GetTask(ctx, mapping.TaskID, "")
	if err != nil {
		return err
	}

	if task.Status == "complete" {
		_, err = h.service.taskService.UpdateTaskStatus(ctx, mapping.TaskID, "todo", "")
		return err
	}

	return nil
}

func (h *IssuesEventHandler) handleEditedIssue(ctx context.Context, _ *domain.GitHubIntegration, issue *github.Issue, mapping *domain.GitHubIssueMapping) error {
	if mapping.SyncDirection == syncDirectionToGitHub {
		return nil // Don't sync back to task
	}

	// Update task with issue changes
	updateReq := domain.UpdateTaskRequest{
		Title:       &[]string{issue.GetTitle()}[0],
		Description: &[]string{issue.GetBody()}[0],
	}

	_, err := h.service.taskService.UpdateTask(ctx, mapping.TaskID, updateReq, "")
	return err
}

// IssueCommentEventHandler handles issue comment events
type IssueCommentEventHandler struct {
	service *GitHubWebhookService
}

// EventType returns the GitHub event type this handler processes
func (h *IssueCommentEventHandler) EventType() string { return "issue_comment" }

// Handle processes GitHub issue comment events
func (h *IssueCommentEventHandler) Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error {
	event, ok := payload.(*github.IssueCommentEvent)
	if !ok {
		return fmt.Errorf("invalid payload type for issue comment event")
	}

	// Check if this issue is linked to a task
	mapping, err := h.service.githubService.issueMappingRepo.GetByIssueNumber(ctx, integration.ID, event.GetIssue().GetNumber())
	if err != nil {
		return nil // No mapping found
	}

	// For now, just log the comment - in a full implementation you might sync comments
	fmt.Printf("Comment on linked issue %d for task %s: %s\n",
		event.GetIssue().GetNumber(),
		mapping.TaskID,
		event.GetComment().GetBody())

	return nil
}

// PullRequestReviewEventHandler handles pull request review events
type PullRequestReviewEventHandler struct {
	service *GitHubWebhookService
}

// EventType returns the GitHub event type this handler processes
func (h *PullRequestReviewEventHandler) EventType() string { return "pull_request_review" }

// Handle processes GitHub pull request review events
func (h *PullRequestReviewEventHandler) Handle(ctx context.Context, integration *domain.GitHubIntegration, payload interface{}) error {
	event, ok := payload.(*github.PullRequestReviewEvent)
	if !ok {
		return fmt.Errorf("invalid payload type for pull request review event")
	}

	// Check if this PR is linked to a task
	mapping, err := h.service.githubService.prMappingRepo.GetByPRNumber(ctx, integration.ID, event.GetPullRequest().GetNumber())
	if err != nil {
		return nil // No mapping found
	}

	review := event.GetReview()
	action := event.GetAction()

	// Handle review actions
	if action == "submitted" {
		switch review.GetState() {
		case "approved":
			// PR approved - potentially move task to review complete
			return h.handleApprovedReview(ctx, mapping)
		case "changes_requested":
			// Changes requested - potentially move task back to development
			return h.handleChangesRequested(ctx, mapping)
		}
	}

	return nil
}

func (h *PullRequestReviewEventHandler) handleApprovedReview(ctx context.Context, mapping *domain.GitHubPRMapping) error {
	task, err := h.service.taskService.GetTask(ctx, mapping.TaskID, "")
	if err != nil {
		return err
	}

	// Move to review complete or ready for merge
	if task.Status == "review" {
		// You might want a "review-approved" status
		// For now, keep in review status but could add metadata later
		return nil
	}

	return nil
}

func (h *PullRequestReviewEventHandler) handleChangesRequested(ctx context.Context, mapping *domain.GitHubPRMapping) error {
	task, err := h.service.taskService.GetTask(ctx, mapping.TaskID, "")
	if err != nil {
		return err
	}

	// Move back to development
	if task.Status == "review" {
		_, err = h.service.taskService.UpdateTaskStatus(ctx, mapping.TaskID, "developing", "")
		return err
	}

	return nil
}
