// Package validation provides comprehensive validation utilities.
package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"simple-easy-tasks/internal/domain"
)

// ValidationError represents a validation error with field-specific details.
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

// ValidationResult represents the result of validation.
type ValidationResult struct {
	Valid  bool               `json:"valid"`
	Errors []*ValidationError `json:"errors,omitempty"`
}

// Validator provides validation functionality.
type Validator struct {
	rules map[string][]ValidationRule
}

// ValidationRule represents a validation rule for a field.
type ValidationRule struct {
	Tag     string
	Message string
	Func    ValidationFunc
}

// ValidationFunc is a function that validates a field value.
type ValidationFunc func(value interface{}, param string) bool

// NewValidator creates a new validator instance.
func NewValidator() *Validator {
	v := &Validator{
		rules: make(map[string][]ValidationRule),
	}
	v.registerBuiltinRules()
	return v
}

// Validate validates a struct using reflection and validation tags.
func (v *Validator) Validate(s interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]*ValidationError, 0),
	}

	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get validation tag
		tag := fieldType.Tag.Get("validate")
		if tag == "" {
			continue
		}

		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		errors := v.validateField(fieldName, field.Interface(), tag)
		result.Errors = append(result.Errors, errors...)
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// validateField validates a single field value against validation rules.
func (v *Validator) validateField(fieldName string, value interface{}, tag string) []*ValidationError {
	errors := make([]*ValidationError, 0)

	rules := parseValidationTag(tag)
	for _, rule := range rules {
		if !v.validateRule(value, rule) {
			errors = append(errors, &ValidationError{
				Field:   fieldName,
				Value:   value,
				Tag:     rule.Tag,
				Message: v.getErrorMessage(fieldName, rule.Tag, rule.Param),
			})
		}
	}

	return errors
}

// ValidationTag represents a parsed validation tag.
type ValidationTag struct {
	Tag   string
	Param string
}

// parseValidationTag parses a validation tag into individual rules.
func parseValidationTag(tag string) []ValidationTag {
	rules := make([]ValidationTag, 0)
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			rules = append(rules, ValidationTag{
				Tag:   kv[0],
				Param: kv[1],
			})
		} else {
			rules = append(rules, ValidationTag{
				Tag:   part,
				Param: "",
			})
		}
	}

	return rules
}

// validateRule validates a value against a single rule.
func (v *Validator) validateRule(value interface{}, rule ValidationTag) bool {
	switch rule.Tag {
	case "required":
		return v.validateRequired(value)
	case "min":
		return v.validateMin(value, rule.Param)
	case "max":
		return v.validateMax(value, rule.Param)
	case "email":
		return v.validateEmail(value)
	case "url":
		return v.validateURL(value)
	case "alpha":
		return v.validateAlpha(value)
	case "alphanum":
		return v.validateAlphaNum(value)
	case "numeric":
		return v.validateNumeric(value)
	case "slug":
		return v.validateSlug(value)
	case "oneof":
		return v.validateOneOf(value, rule.Param)
	default:
		return true // Unknown rules pass
	}
}

// registerBuiltinRules registers built-in validation rules.
func (v *Validator) registerBuiltinRules() {
	// Rules are implemented directly in validateRule for simplicity
}

// Built-in validation functions
func (v *Validator) validateRequired(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case *string:
		return v != nil && strings.TrimSpace(*v) != ""
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case bool:
		return true
	default:
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			return val.Len() > 0
		default:
			return !val.IsZero()
		}
	}
}

func (v *Validator) validateMin(value interface{}, param string) bool {
	minVal, err := strconv.Atoi(param)
	if err != nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return len(v) >= minVal
	case *string:
		if v == nil {
			return true
		}
		return len(*v) >= minVal
	case int:
		return v >= minVal
	case int64:
		return v >= int64(minVal)
	case float64:
		return v >= float64(minVal)
	default:
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return val.Len() >= minVal
		default:
			return true
		}
	}
}

func (v *Validator) validateMax(value interface{}, param string) bool {
	maxVal, err := strconv.Atoi(param)
	if err != nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return len(v) <= maxVal
	case *string:
		if v == nil {
			return true
		}
		return len(*v) <= maxVal
	case int:
		return v <= maxVal
	case int64:
		return v <= int64(maxVal)
	case float64:
		return v <= float64(maxVal)
	default:
		val := reflect.ValueOf(value)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return val.Len() <= maxVal
		default:
			return true
		}
	}
}

func (v *Validator) validateEmail(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true // Empty is valid unless required
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(str)
}

func (v *Validator) validateURL(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true // Empty is valid unless required
	}

	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(str)
}

func (v *Validator) validateAlpha(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true
	}

	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	return alphaRegex.MatchString(str)
}

func (v *Validator) validateAlphaNum(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true
	}

	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return alphaNumRegex.MatchString(str)
}

func (v *Validator) validateNumeric(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true
	}

	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

func (v *Validator) validateSlug(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true
	}

	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(str)
}

func (v *Validator) validateOneOf(value interface{}, param string) bool {
	str, ok := value.(string)
	if !ok {
		if ptr, ok := value.(*string); ok && ptr != nil {
			str = *ptr
		} else {
			return true
		}
	}

	if str == "" {
		return true
	}

	options := strings.Split(param, " ")
	for _, option := range options {
		if str == option {
			return true
		}
	}

	return false
}

// getErrorMessage returns an appropriate error message for a validation rule.
func (v *Validator) getErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only alphabetic characters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a valid number", field)
	case "slug":
		return fmt.Sprintf("%s must be a valid slug (lowercase letters, numbers, and hyphens)", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// ValidateStruct is a convenience function to validate a struct.
func ValidateStruct(s interface{}) *domain.Error {
	validator := NewValidator()
	result := validator.Validate(s)

	if !result.Valid {
		details := make(map[string]interface{})
		for _, err := range result.Errors {
			details[err.Field] = err.Message
		}

		return domain.NewValidationError("VALIDATION_FAILED", "Validation failed", details)
	}

	return nil
}
