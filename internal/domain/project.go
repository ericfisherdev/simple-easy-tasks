package domain

import "time"

// ProjectStatus represents the status of a project.
type ProjectStatus string

const (
	// ActiveProject represents an active project.
	ActiveProject ProjectStatus = "active"
	// ArchivedProject represents an archived project.
	ArchivedProject ProjectStatus = "archived"
)

// ProjectSettings represents project-specific settings.
type ProjectSettings struct {
	CustomFields   map[string]string `json:"custom_fields"`
	Notifications  map[string]bool   `json:"notifications"`
	IsPrivate      bool              `json:"is_private"`
	AllowGuestView bool              `json:"allow_guest_view"`
	EnableComments bool              `json:"enable_comments"`
}

// Project represents a project in the system following DDD principles.
type Project struct {
	UpdatedAt   time.Time       `json:"updated_at"`
	CreatedAt   time.Time       `json:"created_at"`
	Owner       *User           `json:"owner,omitempty"`
	Settings    ProjectSettings `json:"settings"`
	Title       string          `json:"title"`
	ID          string          `json:"id"`
	Description string          `json:"description,omitempty"`
	Slug        string          `json:"slug"`
	OwnerID     string          `json:"owner_id"`
	Color       string          `json:"color,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Status      ProjectStatus   `json:"status"`
	MemberIDs   []string        `json:"member_ids"`
	Members     []User          `json:"members,omitempty"`
}

// IsOwner returns true if the given user is the owner of the project.
func (p *Project) IsOwner(userID string) bool {
	return p.OwnerID == userID
}

// IsMember returns true if the given user is a member of the project.
func (p *Project) IsMember(userID string) bool {
	for _, memberID := range p.MemberIDs {
		if memberID == userID {
			return true
		}
	}
	return false
}

// HasAccess returns true if the given user has access to the project.
func (p *Project) HasAccess(userID string) bool {
	return p.IsOwner(userID) || p.IsMember(userID) || (!p.Settings.IsPrivate && p.Settings.AllowGuestView)
}

// AddMember adds a member to the project if not already a member.
func (p *Project) AddMember(userID string) {
	if !p.IsMember(userID) && !p.IsOwner(userID) {
		p.MemberIDs = append(p.MemberIDs, userID)
		p.UpdatedAt = time.Now()
	}
}

// RemoveMember removes a member from the project.
func (p *Project) RemoveMember(userID string) {
	for i, memberID := range p.MemberIDs {
		if memberID == userID {
			p.MemberIDs = append(p.MemberIDs[:i], p.MemberIDs[i+1:]...)
			p.UpdatedAt = time.Now()
			break
		}
	}
}

// Validate validates the project data.
func (p *Project) Validate() error {
	if err := ValidateRequired("title", p.Title, "INVALID_TITLE", "Title is required"); err != nil {
		return err
	}

	if err := ValidateRequired("slug", p.Slug, "INVALID_SLUG", "Slug is required"); err != nil {
		return err
	}

	if err := ValidateRequired("owner_id", p.OwnerID, "INVALID_OWNER", "Owner ID is required"); err != nil {
		return err
	}

	if err := ValidateEnum("status", string(p.Status), "INVALID_STATUS", "Status must be 'active' or 'archived'",
		string(ActiveProject), string(ArchivedProject)); err != nil {
		return err
	}

	return nil
}

// CreateProjectRequest represents the data needed to create a new project.
type CreateProjectRequest struct {
	Settings    *ProjectSettings `json:"settings,omitempty"`
	Title       string           `json:"title" binding:"required,min=1,max=200"`
	Description string           `json:"description,omitempty"`
	Slug        string           `json:"slug" binding:"required,min=1,max=100"`
	Color       string           `json:"color,omitempty"`
	Icon        string           `json:"icon,omitempty"`
}

// UpdateProjectRequest represents the data that can be updated for a project.
type UpdateProjectRequest struct {
	Title       *string          `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Description *string          `json:"description,omitempty"`
	Color       *string          `json:"color,omitempty"`
	Icon        *string          `json:"icon,omitempty"`
	Settings    *ProjectSettings `json:"settings,omitempty"`
	Status      *ProjectStatus   `json:"status,omitempty"`
}
