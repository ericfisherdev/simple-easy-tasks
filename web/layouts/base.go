// web/layouts/base.go - Base layout component
package layouts

import (
	"github.com/maddalax/htmgo/framework/h"
	"simple-easy-tasks/web/components"
)

func BaseLayout(ctx *h.RequestContext, title string, content *h.Element) *h.Element {
	return h.Html(
		h.Lang("en"),
		h.Class("h-full bg-gray-50"),
		h.Head(
			h.Meta("charset", "UTF-8"),
			h.Meta("viewport", "width=device-width, initial-scale=1.0"),
			h.Title(title+" - Simple Easy Tasks"),

			// Tailwind CSS with HTMGO integration
			h.Link("/static/css/app.css", "stylesheet"),

			// HTMX with HTMGO optimizations
			h.Script("https://unpkg.com/htmx.org@1.9.10"),

			// Alpine.js for client-side interactivity
			h.Script("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js", h.Defer()),

			// GSAP for animations
			h.Script("https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.2/gsap.min.js"),
			h.Script("https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.2/Draggable.min.js"),

			// CSRF token meta tag
			h.Meta("csrf-token", "placeholder-csrf-token"), // ctx.GetCSRFToken() - placeholder for now
		),
		h.Body(
			h.Class("h-full"),
			h.Attribute("x-data", "appData()"),
			h.Div(
				h.Class("min-h-full"),

				// Navigation component
				NavigationComponent(ctx),

				// Main content area
				h.Main(
					h.Class("flex-1"),
					content,
				),

				// Global notifications
				NotificationContainer(ctx),

				// Modal container
				ModalContainer(ctx),
			),

			// Application JavaScript
			h.Script("/static/js/app.js"),
		),
	)
}

// Navigation component placeholder
func NavigationComponent(ctx *h.RequestContext) *h.Element {
	user := &components.User{ID: 1, Name: "Test User", Email: "test@example.com"}
	return components.NavigationComponent(ctx, user)
}

// Notification container for toast messages
func NotificationContainer(ctx *h.RequestContext) *h.Element {
	return h.Div(
		h.Id("notification-container"),
		h.Class("fixed inset-0 flex items-end justify-center px-4 py-6 pointer-events-none sm:p-6 sm:items-start sm:justify-end"),
		h.Attribute("x-data", "{ notifications: [] }"),
		h.Attribute("hx-sse", "connect:/api/notifications/stream"),
		h.Attribute("hx-sse", "message:addNotification"),
	)
}

// Modal container for dynamic modals
func ModalContainer(ctx *h.RequestContext) *h.Element {
	return h.Div(
		h.Id("modal-container"),
		h.Class("relative z-50"),
		h.Attribute("x-data", "modalManager()"),
	)
}
