# Validation Design Decision

## Approach: Side-Effect Free Validation

The `Validate()` methods in the domain layer are designed to be **side-effect free** (immutable). This means:

1. **Validation does not mutate data** - The `Validate()` method only checks values, it doesn't modify them
2. **Normalization happens at boundaries** - Input normalization (trimming whitespace, etc.) occurs in:
   - Constructors (e.g., `NewTask`, `NewComment`)
   - Setter methods when they exist
   - API/service layer before creating domain objects

## Benefits

- **Predictable behavior** - Calling `Validate()` multiple times has no cumulative effect
- **Easier testing** - Tests can validate objects without worrying about mutations
- **Clear separation of concerns** - Validation logic is separate from data transformation
- **Thread safety** - Multiple goroutines can safely validate the same object

## Implementation Pattern

```go
// Constructor normalizes inputs
func NewTask(title, description, projectID, reporterID string) *Task {
    // Normalize inputs at creation
    title = strings.TrimSpace(title)
    description = strings.TrimSpace(description)
    // ... create and return task
}

// Validation checks but doesn't modify
func (t *Task) validateRequiredFields() error {
    if strings.TrimSpace(t.Title) == "" {  // Check trimmed value
        return NewValidationError("title", "Title is required", nil)
    }
    // ... more validation
}
```

This ensures data quality while maintaining clean, predictable validation methods.