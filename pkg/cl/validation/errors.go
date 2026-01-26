package validation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidationError represents a single validation error for a field or key.
type ValidationError struct {
	Field   string         // Field name (for UI mapping)
	Rule    string         // Rule that was violated (e.g., "Required", "MaxLength")
	Message string         // Human-readable message
	Params  map[string]any // Rule parameters (e.g., {"max": 100})
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors that can be accumulated.
type ValidationErrors []ValidationError

// Error implements the error interface, combining all error messages.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Add appends a validation error to the collection.
func (e *ValidationErrors) Add(field, message string) {
	*e = append(*e, ValidationError{Field: field, Message: message})
}

// AddError appends a ValidationError to the collection.
func (e *ValidationErrors) AddError(err ValidationError) {
	*e = append(*e, err)
}

// Merge combines another ValidationErrors into this collection.
func (e *ValidationErrors) Merge(other ValidationErrors) {
	*e = append(*e, other...)
}

// ForField returns all errors for a specific field.
func (e ValidationErrors) ForField(field string) []string {
	var messages []string
	for _, err := range e {
		if err.Field == field {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

// Fields returns all unique field names that have errors.
func (e ValidationErrors) Fields() []string {
	seen := make(map[string]bool)
	var fields []string
	for _, err := range e {
		if err.Field != "" && !seen[err.Field] {
			seen[err.Field] = true
			fields = append(fields, err.Field)
		}
	}
	return fields
}

// ByField returns the first error message for a specific field, or empty string.
func (e ValidationErrors) ByField(field string) string {
	for _, err := range e {
		if err.Field == field {
			return err.Message
		}
	}
	return ""
}

// First returns the first ValidationError, or empty if none.
func (e ValidationErrors) First() ValidationError {
	if len(e) > 0 {
		return e[0]
	}
	return ValidationError{}
}

// AsMap returns errors as a map of field name to slice of messages.
func (e ValidationErrors) AsMap() map[string][]string {
	result := make(map[string][]string)
	for _, err := range e {
		result[err.Field] = append(result[err.Field], err.Message)
	}
	return result
}

// NewSingleError creates a ValidationErrors with a single error.
func NewSingleError(field, message string) ValidationErrors {
	return ValidationErrors{{Field: field, Message: message}}
}

// NewError creates a ValidationErrors with a single general error.
func NewError(message string) ValidationErrors {
	return ValidationErrors{{Message: message}}
}

// --- Predicate functions ---

// IsRequired checks if a string is not empty.
func IsRequired(value string) bool {
	return strings.TrimSpace(value) != ""
}

// IsRequiredUUID checks if a UUID is not the zero UUID.
func IsRequiredUUID(value uuid.UUID) bool {
	return value != uuid.Nil
}

// MinLength checks if a string has at least the minimum length.
func MinLength(value string, min int) bool {
	return len(value) >= min
}

// MaxLength checks if a string does not exceed the maximum length.
func MaxLength(value string, max int) bool {
	return len(value) <= max
}

// --- Validator functions ---

// RequiredString validates that a string field is not empty.
func RequiredString(field, value string) ValidationError {
	if !IsRequired(value) {
		return ValidationError{Field: field, Message: "is required"}
	}
	return ValidationError{}
}

// RequiredUUID validates that a UUID field is not nil.
func RequiredUUID(field string, value uuid.UUID) ValidationError {
	if !IsRequiredUUID(value) {
		return ValidationError{Field: field, Message: "is required"}
	}
	return ValidationError{}
}

// StringMinLength validates that a string has at least the minimum length.
func StringMinLength(field, value string, min int) ValidationError {
	if !MinLength(value, min) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at least %d characters", min)}
	}
	return ValidationError{}
}

// StringMaxLength validates that a string does not exceed the maximum length.
func StringMaxLength(field, value string, max int) ValidationError {
	if !MaxLength(value, max) {
		return ValidationError{Field: field, Message: fmt.Sprintf("must be at most %d characters", max)}
	}
	return ValidationError{}
}
