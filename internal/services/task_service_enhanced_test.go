package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"simple-easy-tasks/internal/domain"
	"simple-easy-tasks/internal/repository"
	"simple-easy-tasks/internal/testutil"
)

// TestTaskService_EnhancedFeatures focuses on testing the new enhanced functionality
// implemented for Week 6: MoveTask, GetProjectTasksFiltered, GetSubtasks,
// GetTaskDependencies, DuplicateTask, and CreateFromTemplate
func TestTaskService_EnhancedFeatures(t *testing.T) {
	// Setup test data
	ctx := context.Background()
	
	// Initialize repositories
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()
	
	// Create service
	service := NewTaskService(taskRepo, projectRepo, userRepo)
	
	// Create test users
	owner := &domain.User{
		ID:       "owner-123",
		Email:    "owner@test.com",
		Username: "owner",
		Role:     domain.RegularUserRole,
	}
	assignee := &domain.User{
		ID:       "assignee-123",
		Email:    "assignee@test.com", 
		Username: "assignee",
		Role:     domain.RegularUserRole,
	}
	unauthorizedUser := &domain.User{
		ID:       "unauthorized-123",
		Email:    "unauthorized@test.com",
		Username: "unauthorized", 
		Role:     domain.RegularUserRole,
	}
	
	userRepo.AddUser(owner)
	userRepo.AddUser(assignee)
	userRepo.AddUser(unauthorizedUser)
	
	// Create test project
	project := &domain.Project{
		ID:        "project-123",
		Title:     "Test Project",
		Slug:      "test-project",
		OwnerID:   owner.ID,
		MemberIDs: []string{assignee.ID},
		Status:    domain.ActiveProject,
		Settings: domain.ProjectSettings{
			IsPrivate:      false,
			AllowGuestView: true,
		},
	}
	projectRepo.AddProject(project)

	t.Run("MoveTask", func(t *testing.T) {
		t.Run("Success_ValidMove", func(t *testing.T) {
			// Create a task in TODO status
			task := &domain.Task{
				ID:         "task-move-1",
				Title:      "Task to Move",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
				Position:   1,
			}
			taskRepo.AddTask(task)
			
			// Move task to developing
			req := MoveTaskRequest{
				TaskID:      task.ID,
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 2,
				ProjectID:   project.ID,
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.NoError(t, err)
			
			// Verify move was called on repository
			assert.True(t, taskRepo.MoveCallLog[task.ID])
		})
		
		t.Run("Error_EmptyTaskID", func(t *testing.T) {
			req := MoveTaskRequest{
				TaskID:      "",
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 1,
				ProjectID:   project.ID,
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Task ID cannot be empty")
		})
		
		t.Run("Error_InvalidStatus", func(t *testing.T) {
			req := MoveTaskRequest{
				TaskID:      "task-1",
				NewStatus:   "invalid-status",
				NewPosition: 1,
				ProjectID:   project.ID,
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Invalid task status")
		})
		
		t.Run("Error_EmptyProjectID", func(t *testing.T) {
			req := MoveTaskRequest{
				TaskID:      "task-1",
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 1,
				ProjectID:   "",
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Project ID cannot be empty")
		})
		
		t.Run("Error_TaskNotFound", func(t *testing.T) {
			req := MoveTaskRequest{
				TaskID:      "nonexistent-task",
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 1,
				ProjectID:   project.ID,
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Task not found")
		})
		
		t.Run("Error_UnauthorizedAccess", func(t *testing.T) {
			// Create a private project for this test
			privateProject := &domain.Project{
				ID:        "private-project-move",
				Title:     "Private Project",
				Slug:      "private-project-move",
				OwnerID:   owner.ID,
				MemberIDs: []string{assignee.ID},
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			task := &domain.Task{
				ID:         "task-move-2",
				Title:      "Task to Move",
				ProjectID:  privateProject.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
				Position:   1,
			}
			taskRepo.AddTask(task)
			
			req := MoveTaskRequest{
				TaskID:      task.ID,
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 1,
				ProjectID:   privateProject.ID,
			}
			
			err := service.MoveTask(ctx, req, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "You don't have access")
		})
		
		t.Run("Error_ProjectMismatch", func(t *testing.T) {
			// Create another project
			otherProject := &domain.Project{
				ID:        "other-project",
				Title:     "Other Project", 
				Slug:      "other-project",
				OwnerID:   owner.ID,
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate: false,
				},
			}
			projectRepo.AddProject(otherProject)
			
			task := &domain.Task{
				ID:         "task-move-3",
				Title:      "Task in Wrong Project",
				ProjectID:  project.ID, // Task belongs to project-123
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
				Position:   1,
			}
			taskRepo.AddTask(task)
			
			req := MoveTaskRequest{
				TaskID:      task.ID,
				NewStatus:   domain.StatusDeveloping,
				NewPosition: 1,
				ProjectID:   otherProject.ID, // But trying to move in other-project
			}
			
			err := service.MoveTask(ctx, req, owner.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Task does not belong to specified project")
		})
	})

	t.Run("GetProjectTasksFiltered", func(t *testing.T) {
		// Clean repository for this test
		taskRepo.Tasks = make(map[string]*domain.Task)
		
		t.Run("Success_NoFilters", func(t *testing.T) {
			// Create test tasks
			task1 := &domain.Task{
				ID:         "filter-task-1",
				Title:      "Task 1",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
				Priority:   domain.PriorityHigh,
			}
			task2 := &domain.Task{
				ID:         "filter-task-2",
				Title:      "Task 2",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusDeveloping,
				Priority:   domain.PriorityMedium,
				AssigneeID: &assignee.ID,
			}
			taskRepo.AddTask(task1)
			taskRepo.AddTask(task2)
			
			filters := repository.TaskFilters{}
			
			tasks, err := service.GetProjectTasksFiltered(ctx, project.ID, filters, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, tasks, 2)
		})
		
		t.Run("Success_StatusFilter", func(t *testing.T) {
			filters := repository.TaskFilters{
				Status: []domain.TaskStatus{domain.StatusTodo},
			}
			
			tasks, err := service.GetProjectTasksFiltered(ctx, project.ID, filters, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, tasks, 1)
			assert.Equal(t, domain.StatusTodo, tasks[0].Status)
		})
		
		t.Run("Error_EmptyProjectID", func(t *testing.T) {
			filters := repository.TaskFilters{}
			
			tasks, err := service.GetProjectTasksFiltered(ctx, "", filters, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, tasks)
			assert.Contains(t, err.Error(), "Project ID cannot be empty")
		})
		
		t.Run("Error_ProjectNotFound", func(t *testing.T) {
			filters := repository.TaskFilters{}
			
			tasks, err := service.GetProjectTasksFiltered(ctx, "nonexistent-project", filters, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, tasks)
		})
		
		t.Run("Error_UnauthorizedAccess_PrivateProject", func(t *testing.T) {
			// Create private project
			privateProject := &domain.Project{
				ID:        "private-project",
				Title:     "Private Project",
				Slug:      "private-project",
				OwnerID:   owner.ID,
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			filters := repository.TaskFilters{}
			
			tasks, err := service.GetProjectTasksFiltered(ctx, privateProject.ID, filters, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Nil(t, tasks)
			assert.Contains(t, err.Error(), "You don't have access to this project")
		})
	})

	t.Run("GetSubtasks", func(t *testing.T) {
		t.Run("Success_HasSubtasks", func(t *testing.T) {
			parentTask := &domain.Task{
				ID:         "parent-task-1",
				Title:      "Parent Task",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(parentTask)
			
			// Mock subtasks that would be returned by repository
			subtask1 := &domain.Task{
				ID:           "subtask-1",
				Title:        "Subtask 1",
				ProjectID:    project.ID,
				ReporterID:   owner.ID,
				ParentTaskID: &parentTask.ID,
			}
			subtask2 := &domain.Task{
				ID:           "subtask-2", 
				Title:        "Subtask 2",
				ProjectID:    project.ID,
				ReporterID:   owner.ID,
				ParentTaskID: &parentTask.ID,
			}
			
			// Setup mock to return these subtasks
			taskRepo.SubtasksByParent[parentTask.ID] = []*domain.Task{subtask1, subtask2}
			
			subtasks, err := service.GetSubtasks(ctx, parentTask.ID, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, subtasks, 2)
		})
		
		t.Run("Success_NoSubtasks", func(t *testing.T) {
			parentTask := &domain.Task{
				ID:         "parent-task-2",
				Title:      "Parent Task No Subs",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(parentTask)
			
			// No subtasks in mock
			
			subtasks, err := service.GetSubtasks(ctx, parentTask.ID, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, subtasks, 0)
		})
		
		t.Run("Error_EmptyParentID", func(t *testing.T) {
			subtasks, err := service.GetSubtasks(ctx, "", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, subtasks)
			assert.Contains(t, err.Error(), "Parent task ID cannot be empty")
		})
		
		t.Run("Error_ParentTaskNotFound", func(t *testing.T) {
			subtasks, err := service.GetSubtasks(ctx, "nonexistent-parent", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, subtasks)
		})
		
		t.Run("Error_UnauthorizedAccess", func(t *testing.T) {
			// Create a private project for this test
			privateProject := &domain.Project{
				ID:        "private-project-subtasks",
				Title:     "Private Project",
				Slug:      "private-project-subtasks",
				OwnerID:   owner.ID,
				MemberIDs: []string{assignee.ID},
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			parentTask := &domain.Task{
				ID:         "parent-task-3",
				Title:      "Parent Task",
				ProjectID:  privateProject.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(parentTask)
			
			subtasks, err := service.GetSubtasks(ctx, parentTask.ID, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Nil(t, subtasks)
			assert.Contains(t, err.Error(), "You don't have access")
		})
	})

	t.Run("GetTaskDependencies", func(t *testing.T) {
		t.Run("Success_HasDependencies", func(t *testing.T) {
			dependentTask := &domain.Task{
				ID:         "dependent-task-1",
				Title:      "Dependent Task",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(dependentTask)
			
			// Mock dependencies
			dep1 := &domain.Task{
				ID:         "dep-1",
				Title:      "Dependency 1",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
			}
			dep2 := &domain.Task{
				ID:         "dep-2",
				Title:      "Dependency 2", 
				ProjectID:  project.ID,
				ReporterID: owner.ID,
			}
			
			taskRepo.DependenciesByTask[dependentTask.ID] = []*domain.Task{dep1, dep2}
			
			deps, err := service.GetTaskDependencies(ctx, dependentTask.ID, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, deps, 2)
		})
		
		t.Run("Success_NoDependencies", func(t *testing.T) {
			independentTask := &domain.Task{
				ID:         "independent-task-1",
				Title:      "Independent Task",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(independentTask)
			
			deps, err := service.GetTaskDependencies(ctx, independentTask.ID, owner.ID)
			assert.NoError(t, err)
			assert.Len(t, deps, 0)
		})
		
		t.Run("Error_EmptyTaskID", func(t *testing.T) {
			deps, err := service.GetTaskDependencies(ctx, "", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, deps)
			assert.Contains(t, err.Error(), "Task ID cannot be empty")
		})
		
		t.Run("Error_TaskNotFound", func(t *testing.T) {
			deps, err := service.GetTaskDependencies(ctx, "nonexistent-task", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, deps)
		})
		
		t.Run("Error_UnauthorizedAccess", func(t *testing.T) {
			// Create a private project for this test
			privateProject := &domain.Project{
				ID:        "private-project-deps",
				Title:     "Private Project",
				Slug:      "private-project-deps",
				OwnerID:   owner.ID,
				MemberIDs: []string{assignee.ID},
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			task := &domain.Task{
				ID:         "dep-task-unauthorized",
				Title:      "Task with Deps",
				ProjectID:  privateProject.ID,
				ReporterID: owner.ID,
				Status:     domain.StatusTodo,
			}
			taskRepo.AddTask(task)
			
			deps, err := service.GetTaskDependencies(ctx, task.ID, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Nil(t, deps)
			assert.Contains(t, err.Error(), "You don't have access")
		})
	})

	t.Run("DuplicateTask", func(t *testing.T) {
		t.Run("Success_BasicDuplication", func(t *testing.T) {
			originalTask := &domain.Task{
				ID:          "original-task-1",
				Title:       "Original Task",
				Description: "Original description",
				ProjectID:   project.ID,
				ReporterID:  owner.ID,
				Status:      domain.StatusDeveloping,
				Priority:    domain.PriorityHigh,
				AssigneeID:  &assignee.ID,
				Progress:    50,
				TimeSpent:   10.5,
				Tags:        []string{"tag1", "tag2"},
			}
			taskRepo.AddTask(originalTask)
			
			options := DuplicationOptions{
				IncludeSubtasks:   false,
				IncludeComments:   false,
				IncludeAttachments: false,
				ResetProgress:     true,
				ResetTimeSpent:    true,
			}
			
			duplicatedTask, err := service.DuplicateTask(ctx, originalTask.ID, options, owner.ID)
			assert.NoError(t, err)
			assert.NotNil(t, duplicatedTask)
			
			// Verify duplication properties
			assert.NotEqual(t, originalTask.ID, duplicatedTask.ID)
			assert.Equal(t, "Copy of "+originalTask.Title, duplicatedTask.Title)
			assert.Equal(t, originalTask.Description, duplicatedTask.Description)
			assert.Equal(t, domain.StatusTodo, duplicatedTask.Status) // Reset to todo
			assert.Equal(t, originalTask.Priority, duplicatedTask.Priority)
			assert.Equal(t, owner.ID, duplicatedTask.ReporterID) // Creator becomes reporter
			assert.Equal(t, 0, duplicatedTask.Progress)         // Reset
			assert.Equal(t, 0.0, duplicatedTask.TimeSpent)      // Reset
			assert.Equal(t, originalTask.Tags, duplicatedTask.Tags)
		})
		
		t.Run("Success_CustomTitle", func(t *testing.T) {
			originalTask := &domain.Task{
				ID:         "original-task-2",
				Title:      "Original Task 2",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
			}
			taskRepo.AddTask(originalTask)
			
			options := DuplicationOptions{
				NewTitle: "Custom Duplicated Task",
			}
			
			duplicatedTask, err := service.DuplicateTask(ctx, originalTask.ID, options, owner.ID)
			assert.NoError(t, err)
			assert.Equal(t, "Custom Duplicated Task", duplicatedTask.Title)
		})
		
		t.Run("Success_PreserveProgressAndTime", func(t *testing.T) {
			originalTask := &domain.Task{
				ID:        "original-task-3",
				Title:     "Task with Progress",
				ProjectID: project.ID,
				ReporterID: owner.ID,
				Progress:  75,
				TimeSpent: 25.5,
			}
			taskRepo.AddTask(originalTask)
			
			options := DuplicationOptions{
				ResetProgress:  false,
				ResetTimeSpent: false,
			}
			
			duplicatedTask, err := service.DuplicateTask(ctx, originalTask.ID, options, owner.ID)
			assert.NoError(t, err)
			assert.Equal(t, 75, duplicatedTask.Progress)
			assert.Equal(t, 25.5, duplicatedTask.TimeSpent)
		})
		
		t.Run("Success_IncludeSubtasks", func(t *testing.T) {
			parentTask := &domain.Task{
				ID:         "parent-dup-1",
				Title:      "Parent for Duplication",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
			}
			taskRepo.AddTask(parentTask)
			
			// Add subtasks to mock
			subtask1 := &domain.Task{
				ID:           "subtask-dup-1",
				Title:        "Subtask 1",
				ProjectID:    project.ID,
				ReporterID:   owner.ID,
				ParentTaskID: &parentTask.ID,
			}
			subtask2 := &domain.Task{
				ID:           "subtask-dup-2",
				Title:        "Subtask 2",
				ProjectID:    project.ID,
				ReporterID:   owner.ID,
				ParentTaskID: &parentTask.ID,
			}
			taskRepo.SubtasksByParent[parentTask.ID] = []*domain.Task{subtask1, subtask2}
			
			options := DuplicationOptions{
				IncludeSubtasks: true,
			}
			
			duplicatedTask, err := service.DuplicateTask(ctx, parentTask.ID, options, owner.ID)
			assert.NoError(t, err)
			assert.NotNil(t, duplicatedTask)
			
			// Verify subtasks were duplicated (mock tracks this)
			assert.True(t, taskRepo.SubtasksDuplicated[parentTask.ID])
		})
		
		t.Run("Error_EmptyTaskID", func(t *testing.T) {
			options := DuplicationOptions{}
			
			duplicatedTask, err := service.DuplicateTask(ctx, "", options, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, duplicatedTask)
			assert.Contains(t, err.Error(), "Task ID cannot be empty")
		})
		
		t.Run("Error_TaskNotFound", func(t *testing.T) {
			options := DuplicationOptions{}
			
			duplicatedTask, err := service.DuplicateTask(ctx, "nonexistent-task", options, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, duplicatedTask)
		})
		
		t.Run("Error_UnauthorizedAccess", func(t *testing.T) {
			// Create a private project for this test
			privateProject := &domain.Project{
				ID:        "private-project-dup",
				Title:     "Private Project",
				Slug:      "private-project-dup",
				OwnerID:   owner.ID,
				MemberIDs: []string{assignee.ID},
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			originalTask := &domain.Task{
				ID:         "original-unauth",
				Title:      "Original Task",
				ProjectID:  privateProject.ID,
				ReporterID: owner.ID,
			}
			taskRepo.AddTask(originalTask)
			
			options := DuplicationOptions{}
			
			duplicatedTask, err := service.DuplicateTask(ctx, originalTask.ID, options, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Nil(t, duplicatedTask)
			assert.Contains(t, err.Error(), "You don't have access")
		})
	})

	t.Run("CreateFromTemplate", func(t *testing.T) {
		t.Run("Success_BasicTemplate", func(t *testing.T) {
			templateTask := &domain.Task{
				ID:             "template-1",
				Title:          "Bug Report Template",
				Description:    "Template for bug reports",
				ProjectID:      project.ID,
				ReporterID:     owner.ID,
				Priority:       domain.PriorityHigh,
				Tags:           []string{"bug", "template"},
				EffortEstimate: func() *float64 { v := 2.0; return &v }(),
			}
			taskRepo.AddTask(templateTask)
			
			newTask, err := service.CreateFromTemplate(ctx, templateTask.ID, project.ID, assignee.ID)
			assert.NoError(t, err)
			assert.NotNil(t, newTask)
			
			// Verify template properties were copied
			assert.NotEqual(t, templateTask.ID, newTask.ID)
			assert.Equal(t, templateTask.Title, newTask.Title)
			assert.Equal(t, templateTask.Description, newTask.Description)
			assert.Equal(t, project.ID, newTask.ProjectID)
			assert.Equal(t, assignee.ID, newTask.ReporterID)
			assert.Equal(t, domain.StatusTodo, newTask.Status) // Always starts as todo
			assert.Equal(t, templateTask.Priority, newTask.Priority)
			assert.Equal(t, 0, newTask.Progress)   // Reset
			assert.Equal(t, 0.0, newTask.TimeSpent) // Reset
		})
		
		t.Run("Error_EmptyTemplateID", func(t *testing.T) {
			newTask, err := service.CreateFromTemplate(ctx, "", project.ID, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, newTask)
			assert.Contains(t, err.Error(), "Template ID cannot be empty")
		})
		
		t.Run("Error_EmptyProjectID", func(t *testing.T) {
			newTask, err := service.CreateFromTemplate(ctx, "template-1", "", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, newTask)
			assert.Contains(t, err.Error(), "Project ID cannot be empty")
		})
		
		t.Run("Error_ProjectNotFound", func(t *testing.T) {
			newTask, err := service.CreateFromTemplate(ctx, "template-1", "nonexistent-project", owner.ID)
			assert.Error(t, err)
			assert.Nil(t, newTask)
		})
		
		t.Run("Error_TemplateNotFound", func(t *testing.T) {
			newTask, err := service.CreateFromTemplate(ctx, "nonexistent-template", project.ID, owner.ID)
			assert.Error(t, err)
			assert.Nil(t, newTask)
		})
		
		t.Run("Error_UnauthorizedAccess", func(t *testing.T) {
			// Create a private project for this test
			privateProject := &domain.Project{
				ID:        "private-project-template",
				Title:     "Private Project",
				Slug:      "private-project-template",
				OwnerID:   owner.ID,
				MemberIDs: []string{assignee.ID},
				Status:    domain.ActiveProject,
				Settings: domain.ProjectSettings{
					IsPrivate:      true,
					AllowGuestView: false,
				},
			}
			projectRepo.AddProject(privateProject)
			
			templateTask := &domain.Task{
				ID:         "template-unauth",
				Title:      "Template Task",
				ProjectID:  project.ID,
				ReporterID: owner.ID,
			}
			taskRepo.AddTask(templateTask)
			
			newTask, err := service.CreateFromTemplate(ctx, templateTask.ID, privateProject.ID, unauthorizedUser.ID)
			assert.Error(t, err)
			assert.Nil(t, newTask)
			assert.Contains(t, err.Error(), "You don't have access to this project")
		})
	})
}

// TestTaskService_AuthorizationEdgeCases tests specific authorization scenarios
func TestTaskService_AuthorizationEdgeCases(t *testing.T) {
	ctx := context.Background()
	
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()
	service := NewTaskService(taskRepo, projectRepo, userRepo)
	
	// Create users
	owner := &domain.User{ID: "owner", Email: "owner@test.com", Username: "owner"}
	member := &domain.User{ID: "member", Email: "member@test.com", Username: "member"}
	outsider := &domain.User{ID: "outsider", Email: "outsider@test.com", Username: "outsider"}
	
	userRepo.AddUser(owner)
	userRepo.AddUser(member)
	userRepo.AddUser(outsider)
	
	t.Run("PrivateProject_OnlyOwnerAndMembersHaveAccess", func(t *testing.T) {
		privateProject := &domain.Project{
			ID:        "private-proj",
			Title:     "Private Project",
			Slug:      "private",
			OwnerID:   owner.ID,
			MemberIDs: []string{member.ID},
			Settings:  domain.ProjectSettings{IsPrivate: true, AllowGuestView: false},
			Status:    domain.ActiveProject,
		}
		projectRepo.AddProject(privateProject)
		
		task := &domain.Task{
			ID:         "private-task",
			Title:      "Private Task",
			ProjectID:  privateProject.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(task)
		
		// Owner should have access
		_, err := service.GetTask(ctx, task.ID, owner.ID)
		assert.NoError(t, err)
		
		// Member should have access
		_, err = service.GetTask(ctx, task.ID, member.ID)
		assert.NoError(t, err)
		
		// Outsider should NOT have access
		_, err = service.GetTask(ctx, task.ID, outsider.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "You don't have access")
	})
	
	t.Run("PublicProject_AllowsGuestView", func(t *testing.T) {
		publicProject := &domain.Project{
			ID:        "public-proj",
			Title:     "Public Project",
			Slug:      "public",
			OwnerID:   owner.ID,
			MemberIDs: []string{member.ID},
			Settings:  domain.ProjectSettings{IsPrivate: false, AllowGuestView: true},
			Status:    domain.ActiveProject,
		}
		projectRepo.AddProject(publicProject)
		
		task := &domain.Task{
			ID:         "public-task",
			Title:      "Public Task",
			ProjectID:  publicProject.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(task)
		
		// Even outsiders should have access to public projects
		_, err := service.GetTask(ctx, task.ID, outsider.ID)
		assert.NoError(t, err)
	})
}

// TestTaskService_ValidationEdgeCases tests comprehensive input validation
func TestTaskService_ValidationEdgeCases(t *testing.T) {
	ctx := context.Background()
	
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()
	service := NewTaskService(taskRepo, projectRepo, userRepo)
	
	// Setup basic test data
	owner := &domain.User{ID: "owner", Email: "owner@test.com", Username: "owner"}
	userRepo.AddUser(owner)
	
	project := &domain.Project{
		ID:       "project",
		Title:    "Project",
		Slug:     "project",
		OwnerID:  owner.ID,
		Settings: domain.ProjectSettings{IsPrivate: false},
		Status:   domain.ActiveProject,
	}
	projectRepo.AddProject(project)
	
	t.Run("MoveTask_NegativePosition", func(t *testing.T) {
		task := &domain.Task{
			ID:         "task-neg-pos",
			Title:      "Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
			Status:     domain.StatusTodo,
		}
		taskRepo.AddTask(task)
		
		req := MoveTaskRequest{
			TaskID:      task.ID,
			NewStatus:   domain.StatusDeveloping,
			NewPosition: -1, // Invalid negative position
			ProjectID:   project.ID,
		}
		
		// The service doesn't validate negative positions (repository handles it),
		// but the request binding should prevent this at API level
		err := service.MoveTask(ctx, req, owner.ID)
		// In a real scenario, this would be caught by the repository
		// For our mock, we don't implement this validation
		assert.NoError(t, err) // Mock allows this
	})
	
	t.Run("TaskFilters_EmptyButValid", func(t *testing.T) {
		// Empty filters should be valid and return all tasks
		filters := repository.TaskFilters{}
		
		tasks, err := service.GetProjectTasksFiltered(ctx, project.ID, filters, owner.ID)
		assert.NoError(t, err)
		assert.NotNil(t, tasks)
	})
}

// TestTaskService_ErrorHandling tests comprehensive error scenarios
func TestTaskService_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	
	// Create a mock that can simulate repository errors
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()
	service := NewTaskService(taskRepo, projectRepo, userRepo)
	
	owner := &domain.User{ID: "owner", Email: "owner@test.com", Username: "owner"}
	userRepo.AddUser(owner)
	
	project := &domain.Project{
		ID:       "project",
		Title:    "Project",
		Slug:     "project", 
		OwnerID:  owner.ID,
		Settings: domain.ProjectSettings{IsPrivate: false},
		Status:   domain.ActiveProject,
	}
	projectRepo.AddProject(project)
	
	t.Run("Repository_CreateError", func(t *testing.T) {
		originalTask := &domain.Task{
			ID:         "original-err",
			Title:      "Original Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(originalTask)
		
		// Force repository to fail on next create
		taskRepo.ForceCreateError = true
		
		options := DuplicationOptions{}
		
		duplicatedTask, err := service.DuplicateTask(ctx, originalTask.ID, options, owner.ID)
		assert.Error(t, err)
		assert.Nil(t, duplicatedTask)
		assert.Contains(t, err.Error(), "Failed to duplicate task")
		
		// Reset error state
		taskRepo.ForceCreateError = false
	})
	
	t.Run("Repository_GetSubtasksError", func(t *testing.T) {
		parentTask := &domain.Task{
			ID:         "parent-err",
			Title:      "Parent Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(parentTask)
		
		// Force repository to fail on GetSubtasks
		taskRepo.ForceGetSubtasksError = true
		
		subtasks, err := service.GetSubtasks(ctx, parentTask.ID, owner.ID)
		assert.Error(t, err)
		assert.Nil(t, subtasks)
		assert.Contains(t, err.Error(), "Failed to fetch subtasks")
		
		// Reset error state  
		taskRepo.ForceGetSubtasksError = false
	})
	
	t.Run("Repository_GetDependenciesError", func(t *testing.T) {
		task := &domain.Task{
			ID:         "task-deps-err",
			Title:      "Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(task)
		
		// Force repository to fail on GetDependencies
		taskRepo.ForceGetDependenciesError = true
		
		deps, err := service.GetTaskDependencies(ctx, task.ID, owner.ID)
		assert.Error(t, err)
		assert.Nil(t, deps)
		assert.Contains(t, err.Error(), "Failed to fetch task dependencies")
		
		// Reset error state
		taskRepo.ForceGetDependenciesError = false
	})
	
	t.Run("Repository_MoveError", func(t *testing.T) {
		task := &domain.Task{
			ID:         "task-move-err",
			Title:      "Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
			Status:     domain.StatusTodo,
		}
		taskRepo.AddTask(task)
		
		// Force repository to fail on Move
		taskRepo.ForceMoveError = true
		
		req := MoveTaskRequest{
			TaskID:      task.ID,
			NewStatus:   domain.StatusDeveloping,
			NewPosition: 1,
			ProjectID:   project.ID,
		}
		
		err := service.MoveTask(ctx, req, owner.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to move task")
		
		// Reset error state
		taskRepo.ForceMoveError = false
	})
}

// TestTaskService_PerformanceConsiderations tests for potential performance issues
func TestTaskService_PerformanceConsiderations(t *testing.T) {
	t.Run("LargeDuplication_SubtaskHandling", func(t *testing.T) {
		// This test would verify that large subtask trees are handled efficiently
		// In a real implementation, we'd want to test pagination, batch processing, etc.
		
		ctx := context.Background()
		taskRepo := testutil.NewMockTaskRepository()
		projectRepo := testutil.NewMockProjectRepository()
		userRepo := testutil.NewMockUserRepository()
		service := NewTaskService(taskRepo, projectRepo, userRepo)
		
		owner := &domain.User{ID: "owner", Email: "owner@test.com", Username: "owner"}
		userRepo.AddUser(owner)
		
		project := &domain.Project{
			ID:       "project",
			Title:    "Project",
			Slug:     "project",
			OwnerID:  owner.ID,
			Settings: domain.ProjectSettings{IsPrivate: false},
			Status:   domain.ActiveProject,
		}
		projectRepo.AddProject(project)
		
		parentTask := &domain.Task{
			ID:         "parent-perf",
			Title:      "Parent Task",
			ProjectID:  project.ID,
			ReporterID: owner.ID,
		}
		taskRepo.AddTask(parentTask)
		
		// Simulate many subtasks
		var subtasks []*domain.Task
		for i := 0; i < 50; i++ {
			subtask := &domain.Task{
				ID:           fmt.Sprintf("subtask-%d", i),
				Title:        fmt.Sprintf("Subtask %d", i),
				ProjectID:    project.ID,
				ReporterID:   owner.ID,
				ParentTaskID: &parentTask.ID,
			}
			subtasks = append(subtasks, subtask)
		}
		taskRepo.SubtasksByParent[parentTask.ID] = subtasks
		
		options := DuplicationOptions{IncludeSubtasks: true}
		
		// This should handle many subtasks without issue
		duplicatedTask, err := service.DuplicateTask(ctx, parentTask.ID, options, owner.ID)
		assert.NoError(t, err)
		assert.NotNil(t, duplicatedTask)
	})
}

// TestTaskService_Integration tests the service with more realistic data
func TestTaskService_Integration(t *testing.T) {
	ctx := context.Background()
	
	taskRepo := testutil.NewMockTaskRepository()
	projectRepo := testutil.NewMockProjectRepository()
	userRepo := testutil.NewMockUserRepository()
	service := NewTaskService(taskRepo, projectRepo, userRepo)
	
	// Create realistic test data
	productOwner := &domain.User{ID: "po-1", Email: "po@company.com", Username: "product_owner"}
	developer1 := &domain.User{ID: "dev-1", Email: "dev1@company.com", Username: "dev_one"}
	developer2 := &domain.User{ID: "dev-2", Email: "dev2@company.com", Username: "dev_two"}
	tester := &domain.User{ID: "tester-1", Email: "tester@company.com", Username: "tester"}
	
	userRepo.AddUser(productOwner)
	userRepo.AddUser(developer1)
	userRepo.AddUser(developer2)
	userRepo.AddUser(tester)
	
	project := &domain.Project{
		ID:        "web-app-proj",
		Title:     "Web Application",
		Slug:      "web-app",
		OwnerID:   productOwner.ID,
		MemberIDs: []string{developer1.ID, developer2.ID, tester.ID},
		Settings:  domain.ProjectSettings{IsPrivate: false, EnableComments: true},
		Status:    domain.ActiveProject,
	}
	projectRepo.AddProject(project)
	
	t.Run("CompleteWorkflow_TaskMovementAndFiltering", func(t *testing.T) {
		// Create initial tasks in backlog
		backlogTask := &domain.Task{
			ID:         "story-1",
			Title:      "User Authentication Feature",
			Description: "Implement login/logout functionality",
			ProjectID:  project.ID,
			ReporterID: productOwner.ID,
			Status:     domain.StatusBacklog,
			Priority:   domain.PriorityHigh,
		}
		taskRepo.AddTask(backlogTask)
		
		// Move to TODO
		moveReq := MoveTaskRequest{
			TaskID:      backlogTask.ID,
			NewStatus:   domain.StatusTodo,
			NewPosition: 1,
			ProjectID:   project.ID,
		}
		err := service.MoveTask(ctx, moveReq, productOwner.ID)
		require.NoError(t, err)
		
		// Assign and move to developing
		moveReq.NewStatus = domain.StatusDeveloping
		moveReq.NewPosition = 1
		err = service.MoveTask(ctx, moveReq, developer1.ID)
		require.NoError(t, err)
		
		// Create subtasks
		backendTask := &domain.Task{
			ID:           "backend-auth",
			Title:        "Backend API for Auth",
			ProjectID:    project.ID,
			ReporterID:   developer1.ID,
			AssigneeID:   &developer1.ID,
			Status:       domain.StatusDeveloping,
			ParentTaskID: &backlogTask.ID,
		}
		frontendTask := &domain.Task{
			ID:           "frontend-auth",
			Title:        "Frontend Auth UI",
			ProjectID:    project.ID,
			ReporterID:   developer1.ID,
			AssigneeID:   &developer2.ID,
			Status:       domain.StatusTodo,
			ParentTaskID: &backlogTask.ID,
		}
		taskRepo.AddTask(backendTask)
		taskRepo.AddTask(frontendTask)
		
		// Mock subtasks relationship
		taskRepo.SubtasksByParent[backlogTask.ID] = []*domain.Task{backendTask, frontendTask}
		
		// Get subtasks
		subtasks, err := service.GetSubtasks(ctx, backlogTask.ID, productOwner.ID)
		require.NoError(t, err)
		assert.Len(t, subtasks, 2)
		
		// Filter by assignee
		filters := repository.TaskFilters{
			AssigneeID: &developer1.ID,
		}
		devTasks, err := service.GetProjectTasksFiltered(ctx, project.ID, filters, developer1.ID)
		require.NoError(t, err)
		assert.Len(t, devTasks, 1)
		assert.Equal(t, backendTask.ID, devTasks[0].ID)
		
		// Filter by status
		filters = repository.TaskFilters{
			Status: []domain.TaskStatus{domain.StatusDeveloping},
		}
		developingTasks, err := service.GetProjectTasksFiltered(ctx, project.ID, filters, productOwner.ID)
		require.NoError(t, err)
		assert.Len(t, developingTasks, 2) // Parent and backend subtask
	})
	
	t.Run("TemplateBasedTaskCreation", func(t *testing.T) {
		// Create a bug report template
		bugTemplate := &domain.Task{
			ID:          "bug-template",
			Title:       "Bug Report",
			Description: "## Steps to Reproduce\n\n## Expected Behavior\n\n## Actual Behavior\n\n## Environment",
			ProjectID:   project.ID,
			ReporterID:  productOwner.ID,
			Priority:    domain.PriorityHigh,
			Tags:        []string{"bug", "needs-investigation"},
		}
		taskRepo.AddTask(bugTemplate)
		
		// Create task from template
		newBugReport, err := service.CreateFromTemplate(ctx, bugTemplate.ID, project.ID, tester.ID)
		require.NoError(t, err)
		
		assert.Equal(t, bugTemplate.Title, newBugReport.Title)
		assert.Equal(t, bugTemplate.Description, newBugReport.Description)
		assert.Equal(t, tester.ID, newBugReport.ReporterID)
		assert.Equal(t, domain.StatusTodo, newBugReport.Status)
		assert.Equal(t, bugTemplate.Priority, newBugReport.Priority)
		assert.Equal(t, bugTemplate.Tags, newBugReport.Tags)
	})
	
	t.Run("TaskDuplicationWithComplexScenario", func(t *testing.T) {
		// Create a complex task with all features
		complexTask := &domain.Task{
			ID:          "complex-feature",
			Title:       "Payment Integration",
			Description: "Integrate Stripe payment processing",
			ProjectID:   project.ID,
			ReporterID:  productOwner.ID,
			AssigneeID:  &developer1.ID,
			Status:      domain.StatusReview,
			Priority:    domain.PriorityCritical,
			Progress:    90,
			TimeSpent:   40.5,
			Tags:        []string{"payment", "integration", "stripe"},
			Dependencies: []string{"auth-feature", "user-management"},
		}
		taskRepo.AddTask(complexTask)
		
		// Duplicate with specific options
		options := DuplicationOptions{
			NewTitle:          "PayPal Integration",
			IncludeSubtasks:   true,
			IncludeComments:   false,
			IncludeAttachments: false,
			ResetProgress:     true,
			ResetTimeSpent:    false, // Keep time estimate
		}
		
		duplicatedTask, err := service.DuplicateTask(ctx, complexTask.ID, options, developer2.ID)
		require.NoError(t, err)
		
		assert.Equal(t, "PayPal Integration", duplicatedTask.Title)
		assert.Equal(t, complexTask.Description, duplicatedTask.Description)
		assert.Equal(t, developer2.ID, duplicatedTask.ReporterID)
		assert.Equal(t, domain.StatusTodo, duplicatedTask.Status)
		assert.Equal(t, 0, duplicatedTask.Progress)      // Reset
		assert.Equal(t, 40.5, duplicatedTask.TimeSpent)  // Preserved
		assert.Equal(t, complexTask.Tags, duplicatedTask.Tags)
		assert.Equal(t, complexTask.Dependencies, duplicatedTask.Dependencies)
	})
}