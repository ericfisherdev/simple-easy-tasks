// web/components/shared.go - Reusable component functions
package components

import (
	"fmt"
	"time"

	"github.com/maddalax/htmgo/framework/h"
)

type User struct {
	ID    int
	Name  string
	Email string
}

// Navigation component with user context
func NavigationComponent(ctx *h.RequestContext, user *User) *h.Element {
	return h.Nav(
		h.Class("bg-white shadow-sm border-b border-gray-200"),
		h.Div(
			h.Class("max-w-7xl mx-auto px-4 sm:px-6 lg:px-8"),
			h.Div(
				h.Class("flex justify-between h-16"),
				// Logo and main navigation
				h.Div(
					h.Class("flex items-center"),
					h.A(
						h.Href("/"),
						h.Class("text-xl font-bold text-primary-600"),
						h.Text("Simple Easy Tasks"),
					),
					// Main nav items
					h.Div(
						h.Class("hidden md:ml-10 md:flex md:space-x-8"),
						NavLink(ctx, "/", "Dashboard"),
						NavLink(ctx, "/projects", "Projects"),
						NavLink(ctx, "/tasks", "Tasks"),
					),
				),
				// User menu
				UserMenuComponent(ctx, user),
			),
		),
	)
}

// Reusable navigation link with active state
func NavLink(ctx *h.RequestContext, href, text string) *h.Element {
	isActive := ctx.Request.URL.Path == href
	baseClass := "text-gray-900 hover:text-primary-600 px-3 py-2 text-sm font-medium"
	activeClass := "text-primary-600 border-b-2 border-primary-600"

	classes := baseClass
	if isActive {
		classes += " " + activeClass
	}

	return h.A(
		h.Href(href),
		h.Class(classes),
		h.Text(text),
	)
}

// User menu component
func UserMenuComponent(ctx *h.RequestContext, user *User) *h.Element {
	return h.Div(
		h.Class("flex items-center space-x-4"),
		h.Text("User: "), // Placeholder for now
		h.Text(user.Name),
	)
}

// Helper functions for common operations
func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)
	if duration.Hours() < 24 {
		return fmt.Sprintf("%.0f hours ago", duration.Hours())
	}
	return fmt.Sprintf("%.0f days ago", duration.Hours()/24)
}

func priorityColor(priority string) string {
	colors := map[string]string{
		"low":      "bg-green-100 text-green-800",
		"medium":   "bg-yellow-100 text-yellow-800",
		"high":     "bg-orange-100 text-orange-800",
		"critical": "bg-red-100 text-red-800",
	}
	return colors[priority]
}

// Placeholder functions for components referenced in code
func getUserProfile(userID int) *h.Element {
	return h.Div(h.Text("User Profile Placeholder"))
}

func getProjectStats(projectID int) *h.Element {
	return h.Div(h.Text("Project Stats Placeholder"))
}
