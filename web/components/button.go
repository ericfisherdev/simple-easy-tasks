// web/components/button.go - Reusable button component
package components

import "github.com/maddalax/htmgo/framework/h"

type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
	ButtonSuccess   ButtonVariant = "success"
)

type ButtonProps struct {
	Text      string
	Icon      string
	Variant   ButtonVariant
	Disabled  bool
	HxGet     string
	HxPost    string
	HxPut     string
	HxDelete  string
	HxTarget  string
	HxSwap    string
	HxTrigger string
	OnClick   string
	Type      string // "button", "submit", "reset"
	Size      string // "sm", "md", "lg"
	Classes   string
}

func Button(props ButtonProps) *h.Element {
	// Base classes
	baseClasses := "inline-flex items-center border border-transparent font-medium rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors duration-200"

	// Size classes
	sizeClasses := map[string]string{
		"sm": "px-3 py-2 text-xs",
		"md": "px-4 py-2 text-sm",
		"lg": "px-6 py-3 text-base",
	}
	size := props.Size
	if size == "" {
		size = "md"
	}

	// Variant classes
	variantClasses := map[ButtonVariant]string{
		ButtonPrimary:   "text-white bg-primary-600 hover:bg-primary-700 focus:ring-primary-500",
		ButtonSecondary: "text-gray-700 bg-white border-gray-300 hover:bg-gray-50 focus:ring-gray-500",
		ButtonDanger:    "text-white bg-red-600 hover:bg-red-700 focus:ring-red-500",
		ButtonSuccess:   "text-white bg-green-600 hover:bg-green-700 focus:ring-green-500",
	}

	classes := baseClasses + " " + sizeClasses[size] + " " + variantClasses[props.Variant]
	if props.Classes != "" {
		classes += " " + props.Classes
	}

	// Build attributes
	attrs := []h.Attributer{
		h.Class(classes),
	}

	// Add type
	buttonType := props.Type
	if buttonType == "" {
		buttonType = "button"
	}
	attrs = append(attrs, h.Type(buttonType))

	// Add disabled state
	if props.Disabled {
		attrs = append(attrs, h.Disabled())
		classes += " opacity-50 cursor-not-allowed"
	}

	// Add HTMX attributes
	if props.HxGet != "" {
		attrs = append(attrs, h.Attribute("hx-get", props.HxGet))
	}
	if props.HxPost != "" {
		attrs = append(attrs, h.Attribute("hx-post", props.HxPost))
	}
	if props.HxPut != "" {
		attrs = append(attrs, h.Attribute("hx-put", props.HxPut))
	}
	if props.HxDelete != "" {
		attrs = append(attrs, h.Attribute("hx-delete", props.HxDelete))
	}
	if props.HxTarget != "" {
		attrs = append(attrs, h.Attribute("hx-target", props.HxTarget))
	}
	if props.HxSwap != "" {
		attrs = append(attrs, h.Attribute("hx-swap", props.HxSwap))
	}
	if props.HxTrigger != "" {
		attrs = append(attrs, h.Attribute("hx-trigger", props.HxTrigger))
	}
	if props.OnClick != "" {
		attrs = append(attrs, h.Attribute("@click", props.OnClick))
	}

	// Build content
	content := []h.Ren{}
	if props.Icon != "" {
		iconClasses := "h-4 w-4"
		if props.Text != "" {
			iconClasses += " -ml-1 mr-2"
		}
		content = append(content, h.I(h.Class(props.Icon+" "+iconClasses)))
	}
	if props.Text != "" {
		content = append(content, h.Text(props.Text))
	}

	return h.Button(attrs...).Children(content...)
}
