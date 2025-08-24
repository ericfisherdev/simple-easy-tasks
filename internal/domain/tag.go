package domain

import (
	"regexp"
	"time"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Tag represents a label that can be associated with tasks
type Tag struct {
	// 8-byte aligned fields first
	CreatedAt time.Time `json:"created_at" db:"created"`
	UpdatedAt time.Time `json:"updated_at" db:"updated"`

	// String fields
	ID        string `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Color     string `json:"color" db:"color"`
	ProjectID string `json:"project_id" db:"project"`

	// 4-byte aligned fields
	UsageCount int `json:"usage_count" db:"usage_count"`
}

// NewTag creates a new tag with default values
func NewTag(name, color, projectID string) *Tag {
	now := time.Now()
	return &Tag{
		Name:       name,
		Color:      color,
		ProjectID:  projectID,
		UsageCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Validate performs comprehensive validation of the tag
func (t *Tag) Validate() error {
	if t.Name == "" {
		return NewValidationError("name", "Tag name is required", nil)
	}
	if len(t.Name) > 50 {
		return NewValidationError("name", "Tag name must not exceed 50 characters", nil)
	}
	if t.Color == "" {
		return NewValidationError("color", "Tag color is required", nil)
	}
	if !isValidHexColor(t.Color) {
		return NewValidationError("color", "Tag color must be a valid hex color", nil)
	}
	if t.ProjectID == "" {
		return NewValidationError("project_id", "Project ID is required", nil)
	}
	if t.UsageCount < 0 {
		return NewValidationError("usage_count", "Usage count cannot be negative", nil)
	}
	return nil
}

// IncrementUsage increases the usage counter by one
func (t *Tag) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

// DecrementUsage decreases the usage counter by one
func (t *Tag) DecrementUsage() {
	if t.UsageCount > 0 {
		t.UsageCount--
		t.UpdatedAt = time.Now()
	}
}

// UpdateName changes the tag name with validation
func (t *Tag) UpdateName(name string) error {
	if name == "" {
		return NewValidationError("name", "Tag name is required", nil)
	}
	if len(name) > 50 {
		return NewValidationError("name", "Tag name must not exceed 50 characters", nil)
	}
	t.Name = name
	t.UpdatedAt = time.Now()
	return nil
}

// UpdateColor changes the tag color with validation
func (t *Tag) UpdateColor(color string) error {
	if color == "" {
		return NewValidationError("color", "Tag color is required", nil)
	}
	if !isValidHexColor(color) {
		return NewValidationError("color", "Tag color must be a valid hex color", nil)
	}
	t.Color = color
	t.UpdatedAt = time.Now()
	return nil
}

func isValidHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}
