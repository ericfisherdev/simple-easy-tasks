package domain

import (
	"testing"
	"time"
)

func TestTask_Archive(t *testing.T) {
	// Create a new task
	task := NewTask("Test Task", "Test Description", "project1", "user1")

	// Verify initial state
	if task.Archived {
		t.Error("Expected task to not be archived initially")
	}
	if task.ArchivedAt != nil {
		t.Error("Expected ArchivedAt to be nil initially")
	}

	// Archive the task
	beforeArchive := time.Now().UTC()
	task.Archive()
	afterArchive := time.Now().UTC()

	// Verify archived state
	if !task.Archived {
		t.Error("Expected task to be archived")
	}
	if task.ArchivedAt == nil {
		t.Fatal("Expected ArchivedAt to be set")
	}
	if task.ArchivedAt.Before(beforeArchive) || task.ArchivedAt.After(afterArchive) {
		t.Error("Expected ArchivedAt to be within the test time range")
	}
}

func TestTask_Unarchive(t *testing.T) {
	// Create and archive a task
	task := NewTask("Test Task", "Test Description", "project1", "user1")
	task.Archive()

	// Verify it's archived
	if !task.Archived {
		t.Fatal("Expected task to be archived after Archive()")
	}

	// Unarchive the task
	task.Unarchive()

	// Verify unarchived state
	if task.Archived {
		t.Error("Expected task to not be archived after Unarchive()")
	}
	if task.ArchivedAt != nil {
		t.Error("Expected ArchivedAt to be nil after Unarchive()")
	}
}

func TestTask_IsArchived(t *testing.T) {
	// Create a new task
	task := NewTask("Test Task", "Test Description", "project1", "user1")

	// Test initial state
	if task.IsArchived() {
		t.Error("Expected IsArchived() to return false for new task")
	}

	// Test archived state
	task.Archive()
	if !task.IsArchived() {
		t.Error("Expected IsArchived() to return true for archived task")
	}

	// Test unarchived state
	task.Unarchive()
	if task.IsArchived() {
		t.Error("Expected IsArchived() to return false for unarchived task")
	}
}
