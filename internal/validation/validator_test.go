package validation_test

//nolint:gofumpt
import (
	"testing"

	"simple-easy-tasks/internal/validation"
)

// TestStruct for validation testing
type TestStruct struct {
	Website  *string `json:"website,omitempty" validate:"url"`
	Name     string  `json:"name" validate:"required,min=2,max=50"`
	Email    string  `json:"email" validate:"required,email"`
	Username string  `json:"username" validate:"required,min=3,max=20,alphanum"`
	Role     string  `json:"role" validate:"required,oneof=admin user guest"`
	Slug     string  `json:"slug" validate:"slug"`
	Age      int     `json:"age" validate:"min=0,max=150"`
}

func TestValidator_ValidateStruct_Success(t *testing.T) {
	website := "https://example.com"
	testData := TestStruct{
		Name:     "John Doe",
		Email:    "john@example.com",
		Username: "johndoe123",
		Age:      30,
		Website:  &website,
		Role:     "user",
		Slug:     "john-doe",
	}

	validator := validation.NewValidator()
	result := validator.Validate(testData)

	if !result.Valid {
		t.Errorf("Expected validation to pass, but got errors: %+v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
	}
}

func TestValidator_ValidateStruct_RequiredFields(t *testing.T) {
	testData := TestStruct{
		Name:     "", // Required but empty
		Email:    "", // Required but empty
		Username: "", // Required but empty
		Age:      25,
		Role:     "user",
		Slug:     "test-slug",
	}

	validator := validation.NewValidator()
	result := validator.Validate(testData)

	if result.Valid {
		t.Error("Expected validation to fail for missing required fields")
	}

	// Debug: Print all errors
	t.Logf("Got %d errors:", len(result.Errors))
	for _, err := range result.Errors {
		t.Logf("  Field: %s, Tag: %s, Message: %s", err.Field, err.Tag, err.Message)
	}

	// Check that at least the required fields have errors
	errorFields := make(map[string]bool)
	for _, err := range result.Errors {
		if err.Tag == "required" {
			errorFields[err.Field] = true
		}
	}

	requiredFields := []string{"name", "email", "username"}
	for _, field := range requiredFields {
		if !errorFields[field] {
			t.Errorf("Expected required error for field: %s", field)
		}
	}
}

func TestValidator_ValidateStruct_EmailValidation(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		shouldErr bool
	}{
		{"valid email", "test@example.com", false},
		{"invalid email - no @", "testexample.com", true},
		{"invalid email - no domain", "test@", true},
		{"invalid email - no TLD", "test@example", true},
		{"empty email", "", true}, // Required field
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := TestStruct{
				Name:     "Test User",
				Email:    tt.email,
				Username: "testuser",
				Age:      25,
				Role:     "user",
				Slug:     "test-slug",
			}

			validator := validation.NewValidator()
			result := validator.Validate(testData)

			if tt.shouldErr && result.Valid {
				t.Errorf("Expected validation to fail for email: %s", tt.email)
			}

			if !tt.shouldErr && !result.Valid {
				t.Errorf("Expected validation to pass for email: %s, but got errors: %+v", tt.email, result.Errors)
			}
		})
	}
}

func TestValidator_ValidateStruct_MinMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		errorTag  string
		shouldErr bool
	}{
		{"valid length", "john123", "", false},
		{"too short", "ab", "min", true},
		{"too long", "thisusernameiswaytoolongtobevalid", "max", true},
		{"minimum valid", "abc", "", false},
		{"maximum valid", "12345678901234567890", "", false}, // exactly 20 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := TestStruct{
				Name:     "Test User",
				Email:    "test@example.com",
				Username: tt.username,
				Age:      25,
				Role:     "user",
				Slug:     "test-slug",
			}

			validator := validation.NewValidator()
			result := validator.Validate(testData)

			if tt.shouldErr && result.Valid {
				t.Errorf("Expected validation to fail for username: %s", tt.username)
			}

			if !tt.shouldErr && !result.Valid {
				t.Errorf("Expected validation to pass for username: %s, but got errors: %+v", tt.username, result.Errors)
			}

			if tt.shouldErr {
				found := false
				for _, err := range result.Errors {
					if err.Field == "username" && err.Tag == tt.errorTag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error tag %s for username field", tt.errorTag)
				}
			}
		})
	}
}

func TestValidator_ValidateStruct_OneOfValidation(t *testing.T) {
	tests := []struct {
		name      string
		role      string
		shouldErr bool
	}{
		{"valid role - admin", "admin", false},
		{"valid role - user", "user", false},
		{"valid role - guest", "guest", false},
		{"invalid role", "superuser", true},
		{"empty role", "", true}, // This will fail required validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := TestStruct{
				Name:     "Test User",
				Email:    "test@example.com",
				Username: "testuser",
				Age:      25,
				Role:     tt.role,
				Slug:     "test-slug",
			}

			validator := validation.NewValidator()
			result := validator.Validate(testData)

			if tt.shouldErr && result.Valid {
				t.Errorf("Expected validation to fail for role: %s", tt.role)
			}

			if !tt.shouldErr && !result.Valid {
				t.Errorf("Expected validation to pass for role: %s, but got errors: %+v", tt.role, result.Errors)
			}
		})
	}
}

func TestValidator_ValidateStruct_SlugValidation(t *testing.T) {
	tests := []struct {
		name      string
		slug      string
		shouldErr bool
	}{
		{"valid slug", "my-test-slug", false},
		{"valid slug with numbers", "test-123-slug", false},
		{"single word", "test", false},
		{"invalid - uppercase", "My-Test-Slug", true},
		{"invalid - spaces", "my test slug", true},
		{"invalid - special chars", "my_test_slug", true},
		{"invalid - starting with dash", "-test-slug", true},
		{"invalid - ending with dash", "test-slug-", true},
		{"invalid - double dash", "test--slug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := TestStruct{
				Name:     "Test User",
				Email:    "test@example.com",
				Username: "testuser",
				Age:      25,
				Role:     "user",
				Slug:     tt.slug,
			}

			validator := validation.NewValidator()
			result := validator.Validate(testData)

			if tt.shouldErr && result.Valid {
				t.Errorf("Expected validation to fail for slug: %s", tt.slug)
			}

			if !tt.shouldErr && !result.Valid {
				t.Errorf("Expected validation to pass for slug: %s, but got errors: %+v", tt.slug, result.Errors)
			}
		})
	}
}

func TestValidator_ValidateStruct_URLValidation(t *testing.T) {
	tests := []struct {
		website   *string
		name      string
		shouldErr bool
	}{
		{stringPtr("https://example.com"), "valid https URL", false},
		{stringPtr("http://example.com"), "valid http URL", false},
		{stringPtr("https://example.com/path"), "valid URL with path", false},
		{stringPtr("example.com"), "invalid URL - no protocol", true},
		{stringPtr("ftp://example.com"), "invalid URL - ftp protocol", true},
		{nil, "nil pointer", false}, // Should pass as it's optional
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testData := TestStruct{
				Name:     "Test User",
				Email:    "test@example.com",
				Username: "testuser",
				Age:      25,
				Role:     "user",
				Slug:     "test-slug",
				Website:  tt.website,
			}

			validator := validation.NewValidator()
			result := validator.Validate(testData)

			if tt.shouldErr && result.Valid {
				websiteStr := "nil"
				if tt.website != nil {
					websiteStr = *tt.website
				}
				t.Errorf("Expected validation to fail for website: %s", websiteStr)
			}

			if !tt.shouldErr && !result.Valid {
				websiteStr := "nil"
				if tt.website != nil {
					websiteStr = *tt.website
				}
				t.Errorf("Expected validation to pass for website: %s, but got errors: %+v", websiteStr, result.Errors)
			}
		})
	}
}

func TestValidateStruct_ConvenienceFunction(t *testing.T) {
	// Test the convenience function
	validData := TestStruct{
		Name:     "John Doe",
		Email:    "john@example.com",
		Username: "johndoe",
		Age:      30,
		Role:     "user",
		Slug:     "john-doe",
	}

	err := validation.ValidateStruct(validData)
	if err != nil {
		t.Errorf("Expected no error for valid data, got: %v", err)
	}

	// Test with invalid data
	invalidData := TestStruct{
		Name:     "", // Required but empty
		Email:    "invalid-email",
		Username: "ab", // Too short
		Age:      30,
		Role:     "invalid-role",
		Slug:     "Invalid-Slug",
	}

	err = validation.ValidateStruct(invalidData)
	if err == nil {
		t.Error("Expected validation error for invalid data")
		return
	}

	// Check if it's a domain validation error
	if err.Type != "VALIDATION_ERROR" {
		t.Errorf("Expected validation error type, got: %s", err.Type)
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
