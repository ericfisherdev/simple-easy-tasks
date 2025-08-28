# Component Library Documentation

## Overview
This component library provides a comprehensive set of UI components for the Simple Easy Tasks application, built with HTMGO, HTMX, and Tailwind CSS.

## Button Variants and States

### Primary Button
```go
Button(ButtonProps{
    Text: "Primary Action",
    Variant: "primary",
    Size: "medium",
})
```
- **Use case**: Main actions like "Create Task", "Save", "Submit"
- **Classes**: `bg-primary-600 hover:bg-primary-700 text-white`
- **States**: Default, hover, focus, disabled, loading

### Secondary Button
```go
Button(ButtonProps{
    Text: "Secondary Action", 
    Variant: "secondary",
    Size: "medium",
})
```
- **Use case**: Secondary actions like "Cancel", "Edit"
- **Classes**: `bg-gray-200 hover:bg-gray-300 text-gray-800`
- **States**: Default, hover, focus, disabled

### Danger Button
```go
Button(ButtonProps{
    Text: "Delete Task",
    Variant: "danger", 
    Size: "medium",
})
```
- **Use case**: Destructive actions like "Delete", "Remove"
- **Classes**: `bg-danger hover:bg-red-600 text-white`
- **States**: Default, hover, focus, disabled, loading

### Button Sizes
- **Small**: `text-sm px-3 py-1.5` - For compact spaces
- **Medium**: `text-base px-4 py-2` - Default size
- **Large**: `text-lg px-6 py-3` - For prominent actions

## Form Input Components

### Text Input
```go
Input(InputProps{
    Type: "text",
    Name: "title",
    Label: "Task Title",
    Placeholder: "Enter task title...",
    Required: true,
})
```
- **Classes**: `border border-gray-300 rounded-md px-3 py-2`
- **States**: Default, focus, error, disabled
- **Validation**: Real-time validation with error messages

### Textarea
```go
Textarea(TextareaProps{
    Name: "description",
    Label: "Description", 
    Rows: 4,
    Placeholder: "Task description...",
})
```
- **Classes**: `border border-gray-300 rounded-md px-3 py-2 resize-vertical`
- **States**: Default, focus, error, disabled

### Select Dropdown
```go
Select(SelectProps{
    Name: "priority",
    Label: "Priority",
    Options: []SelectOption{
        {Value: "low", Label: "Low Priority"},
        {Value: "medium", Label: "Medium Priority"},  
        {Value: "high", Label: "High Priority"},
    },
})
```
- **Classes**: `border border-gray-300 rounded-md px-3 py-2`
- **States**: Default, focus, error, disabled, expanded

## Card and Panel Designs

### Basic Card
```go
Card(CardProps{
    Title: "Task Card",
    Children: []h.Ren{
        P("Task content goes here"),
    },
})
```
- **Classes**: `bg-white rounded-lg border border-gray-200 shadow-card`
- **Variants**: Default, elevated (more shadow), outlined (thicker border)

### Task Card
```go
TaskCard(TaskCardProps{
    Task: task,
    ShowActions: true,
    Draggable: true,
})
```
- **Classes**: `bg-white rounded-lg p-4 border border-gray-200 hover:shadow-hover`
- **Features**: Priority indicator, due date, assignee, drag handle
- **States**: Default, hover, dragging, selected

### Statistics Card  
```go
StatsCard(StatsCardProps{
    Title: "Total Tasks",
    Value: "24",
    Icon: "tasks",
    Trend: "+12%",
    TrendUp: true,
})
```
- **Classes**: `bg-white rounded-lg p-6 border border-gray-200 shadow-card`
- **Features**: Large number display, trend indicators, icons

## Navigation Components

### Main Navigation
```go
MainNav(MainNavProps{
    Items: []NavItem{
        {Label: "Dashboard", URL: "/", Active: true},
        {Label: "Tasks", URL: "/tasks", Badge: "5"},
        {Label: "Projects", URL: "/projects"},
    },
})
```
- **Classes**: `bg-white border-b border-gray-200 px-6 py-4`
- **Features**: Active state, badges, responsive collapse

### Breadcrumbs
```go
Breadcrumbs(BreadcrumbsProps{
    Items: []BreadcrumbItem{
        {Label: "Projects", URL: "/projects"},
        {Label: "My Project", URL: "/projects/123"},
        {Label: "Task Details", Current: true},
    },
})
```
- **Classes**: `flex items-center space-x-2 text-sm text-gray-500`
- **Features**: Separators, current page indicator, link hover states

### Sidebar Navigation
```go
Sidebar(SidebarProps{
    Items: sidebarItems,
    Collapsed: false,
    User: currentUser,
})
```
- **Classes**: `w-64 bg-gray-50 border-r border-gray-200 h-screen`
- **Features**: Collapsible, user profile section, section groupings

## Modal and Overlay Designs

### Basic Modal
```go
Modal(ModalProps{
    Title: "Create New Task",
    Show: true,
    Children: []h.Ren{
        TaskForm(TaskFormProps{}),
    },
})
```
- **Classes**: `fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center`
- **Features**: Backdrop click to close, escape key handling, focus trap

### Confirmation Modal
```go
ConfirmModal(ConfirmModalProps{
    Title: "Delete Task",
    Message: "Are you sure you want to delete this task? This action cannot be undone.",
    ConfirmText: "Delete",
    ConfirmVariant: "danger",
    Show: true,
})
```
- **Classes**: `max-w-md bg-white rounded-lg shadow-modal p-6`
- **Features**: Icon, destructive styling, clear actions

### Toast Notifications
```go
Toast(ToastProps{
    Type: "success", // success, error, warning, info
    Title: "Task Created",
    Message: "Your task has been created successfully.",
    Duration: 5000,
})
```
- **Classes**: `fixed top-4 right-4 max-w-sm bg-white rounded-lg shadow-lg border-l-4`
- **Features**: Auto-dismiss, manual close, stacking, animations

## Icon System and Usage

### Icon Component
```go
Icon(IconProps{
    Name: "plus",
    Size: "md", // xs, sm, md, lg, xl
    Color: "current", // Uses current text color
})
```

### Available Icons
- **Actions**: plus, edit, delete, save, cancel, search
- **Status**: check, x, warning, info, loading
- **Navigation**: arrow-left, arrow-right, arrow-up, arrow-down
- **Objects**: task, project, user, calendar, settings

### Icon Sizes
- **xs**: 12px (w-3 h-3)
- **sm**: 16px (w-4 h-4)  
- **md**: 20px (w-5 h-5) - Default
- **lg**: 24px (w-6 h-6)
- **xl**: 32px (w-8 h-8)

### Usage Guidelines
- Use consistent sizes within the same context
- Pair icons with text for better accessibility
- Use color to convey meaning (red for delete, green for success)
- Provide alt text for screen readers

## Component Composition Examples

### Task Creation Form
```go
Modal(ModalProps{
    Title: "Create New Task",
    Children: []h.Ren{
        Form(FormProps{
            Children: []h.Ren{
                Input(InputProps{
                    Name: "title",
                    Label: "Task Title",
                    Required: true,
                }),
                Textarea(TextareaProps{
                    Name: "description", 
                    Label: "Description",
                }),
                Select(SelectProps{
                    Name: "priority",
                    Label: "Priority", 
                    Options: priorityOptions,
                }),
                Div(
                    Class("flex justify-end space-x-3 mt-6"),
                    Button(ButtonProps{
                        Text: "Cancel",
                        Variant: "secondary",
                        Type: "button",
                    }),
                    Button(ButtonProps{
                        Text: "Create Task",
                        Variant: "primary", 
                        Type: "submit",
                    }),
                ),
            },
        }),
    },
})
```

### Dashboard Stats Grid
```go
Grid(GridProps{
    Columns: 3,
    Gap: "md",
    Children: []h.Ren{
        StatsCard(StatsCardProps{
            Title: "Total Tasks",
            Value: "24",
            Icon: "tasks",
        }),
        StatsCard(StatsCardProps{
            Title: "Completed",
            Value: "18", 
            Icon: "check",
        }),
        StatsCard(StatsCardProps{
            Title: "In Progress",
            Value: "6",
            Icon: "clock",
        }),
    },
})
```

## Accessibility Guidelines

- All interactive elements support keyboard navigation
- Color is not the only means of conveying information
- Form inputs have proper labels and error messages
- Focus indicators are clearly visible
- Screen reader support with proper ARIA attributes
- High contrast ratios meet WCAG AA standards

## Responsive Design

- Mobile-first approach with Tailwind breakpoints
- Components adapt to different screen sizes
- Touch-friendly tap targets (minimum 44px)
- Readable text sizes on all devices
- Proper spacing and layout on mobile