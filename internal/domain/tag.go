package domain

import (
	"time"
)

type Tag struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Color      string    `json:"color" db:"color"`
	ProjectID  string    `json:"project_id" db:"project"`
	UsageCount int       `json:"usage_count" db:"usage_count"`
	CreatedAt  time.Time `json:"created_at" db:"created"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated"`
}

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

func (t *Tag) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

func (t *Tag) DecrementUsage() {
	if t.UsageCount > 0 {
		t.UsageCount--
		t.UpdatedAt = time.Now()
	}
}

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
	if len(color) != 7 {
		return false
	}
	if color[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
