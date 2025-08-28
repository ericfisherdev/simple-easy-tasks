// web/partials/task_card.go - Dynamic task card partial
package partials

import (
	"fmt"
	"strings"

	"github.com/maddalax/htmgo/framework/h"
)

type Task struct {
	ID          int
	Title       string
	Description string
	Priority    string
	Tags        []*Tag
}

type Tag struct {
	ID    int
	Name  string
	Color string
}

func TaskCard(ctx *h.RequestContext, task *Task) *h.Element {
	return h.Div(
		h.Class("task-card bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-3 cursor-pointer hover:shadow-md transition-shadow duration-200"),
		h.Attribute("data-task-id", fmt.Sprintf("%d", task.ID)),
		h.Attribute("x-data", "taskCard($task)"),
		h.Attribute("draggable", "true"),
		h.Attribute("@click", "openTaskDetails()"),
		h.Attribute("@dragstart", "onDragStart($event)"),
		h.Attribute("@dragend", "onDragEnd($event)"),

		// Task header with priority and title
		h.Div(
			h.Class("flex items-start justify-between mb-3"),
			h.Div(
				h.Class("flex items-center flex-1"),
				PriorityIndicator(task.Priority),
				h.H4(
					h.Class("text-sm font-medium text-gray-900 line-clamp-2 ml-2"),
					h.Text(task.Title),
				),
			),
			TaskActions(ctx, task),
		),

		// Description if present
		h.If(task.Description != "",
			h.P(
				h.Class("text-sm text-gray-600 line-clamp-2 mb-3"),
				h.Text(task.Description),
			),
		),

		// Tags
		h.If(len(task.Tags) > 0,
			h.Div(
				h.Class("flex flex-wrap gap-1 mb-3"),
				h.ForEach(task.Tags, func(tag *Tag, index int) *h.Element {
					return TagComponent(tag)
				}),
			),
		),

		// Footer with assignee and metadata
		TaskFooter(task),
	)
}

func PriorityIndicator(priority string) *h.Element {
	colorClass := priorityColor(priority)
	return h.Div(
		h.Class("w-3 h-3 rounded-full flex-shrink-0 "+colorClass),
		h.Attribute("title", strings.Title(priority)+" priority"),
	)
}

func TaskActions(ctx *h.RequestContext, task *Task) *h.Element {
	return h.Div(
		h.Class("flex items-center space-x-2"),
		h.Button(
			h.Class("text-gray-400 hover:text-gray-600"),
			h.Text("Edit"),
		),
		h.Button(
			h.Class("text-red-400 hover:text-red-600"),
			h.Text("Delete"),
		),
	)
}

func TagComponent(tag *Tag) *h.Element {
	return h.Span(
		h.Class("inline-block bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded"),
		h.Text(tag.Name),
	)
}

func TaskFooter(task *Task) *h.Element {
	return h.Div(
		h.Class("flex items-center justify-between text-xs text-gray-500"),
		h.Div(h.Text("Footer placeholder")),
	)
}

func priorityColor(priority string) string {
	colors := map[string]string{
		"low":      "bg-green-100 text-green-800",
		"medium":   "bg-yellow-100 text-yellow-800",
		"high":     "bg-orange-100 text-orange-800",
		"critical": "bg-red-100 text-red-800",
	}
	if color, exists := colors[priority]; exists {
		return color
	}
	return "bg-gray-100 text-gray-800"
}
