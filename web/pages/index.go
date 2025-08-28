// web/pages/index.go - Main dashboard page
package pages

import (
	"time"

	"github.com/maddalax/htmgo/framework/h"
)

func IndexPage(ctx *h.RequestContext) *h.Page {
	// user := getUserFromContext(ctx)
	// projects := getActiveProjects(ctx, user.ID)

	return h.NewPage(
		h.Html(
			h.Head(
				h.Title("Dashboard - Simple Easy Tasks"),
				h.Meta("viewport", "width=device-width, initial-scale=1.0"),
				h.Link("/static/css/app.css", "stylesheet"),
				h.Script("https://unpkg.com/htmx.org@1.9.10"),
				h.Script("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js", h.Defer()),
			),
			h.Body(
				h.Class("min-h-screen bg-gray-50"),
				h.Attribute("x-data", "dashboardData()"),

				// Navigation
				// NavigationComponent(ctx, user),

				// Main content
				h.Main(
					h.Class("max-w-7xl mx-auto py-6 sm:px-6 lg:px-8"),
					h.Div(
						h.Class("px-4 py-6 sm:px-0"),
						h.H1(
							h.Class("text-2xl font-bold text-gray-900 mb-6"),
							h.Text("Welcome back!"), // h.Text(user.Name),
						),

						// Project overview cards
						// ProjectOverviewGrid(ctx, projects),

						// Recent tasks
						// RecentTasksList(ctx, user.ID),
					),
				),
			),
		),
	)
}

// Template data structure for consistency
type TemplateData struct {
	User      *User
	Project   *Project
	PageTitle string
	PageData  interface{}
	CSRFToken string
	Messages  []Message
}

type User struct {
	ID    int
	Name  string
	Email string
}

type Project struct {
	ID          int
	Name        string
	Description string
}

type Message struct {
	Type    string
	Content string
}
