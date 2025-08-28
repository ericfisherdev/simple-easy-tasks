// web/components/card.go - Reusable card component
package components

import "github.com/maddalax/htmgo/framework/h"

type CardProps struct {
	Header   string
	Subtitle string
	Content  *h.Element
	Footer   *h.Element
	Hover    bool
	Classes  string
	OnClick  string
	HxGet    string
	HxTarget string
}

func Card(props CardProps) *h.Element {
	classes := "bg-white overflow-hidden shadow-card rounded-lg"
	if props.Hover {
		classes += " hover:shadow-hover transition-shadow duration-200"
	}
	if props.Classes != "" {
		classes += " " + props.Classes
	}

	attrs := []h.Attributer{h.Class(classes)}

	// Add HTMX attributes
	if props.HxGet != "" {
		attrs = append(attrs, h.Attribute("hx-get", props.HxGet))
	}
	if props.HxTarget != "" {
		attrs = append(attrs, h.Attribute("hx-target", props.HxTarget))
	}
	if props.OnClick != "" {
		attrs = append(attrs, h.Attribute("@click", props.OnClick))
	}

	children := []h.Ren{}

	// Header section
	if props.Header != "" {
		headerContent := []h.Ren{
			h.H3(
				h.Class("text-lg font-medium text-gray-900"),
				h.Text(props.Header),
			),
		}
		if props.Subtitle != "" {
			headerContent = append(headerContent,
				h.P(
					h.Class("mt-1 text-sm text-gray-500"),
					h.Text(props.Subtitle),
				),
			)
		}

		children = append(children,
			h.Div(
				h.Class("px-6 py-4 border-b border-gray-200"),
				headerContent...,
			),
		)
	}

	// Content section
	if props.Content != nil {
		children = append(children,
			h.Div(
				h.Class("px-6 py-4"),
				props.Content,
			),
		)
	}

	// Footer section
	if props.Footer != nil {
		children = append(children,
			h.Div(
				h.Class("px-6 py-3 bg-gray-50 border-t border-gray-200"),
				props.Footer,
			),
		)
	}

	return h.Div(attrs...).Children(children...)
}
