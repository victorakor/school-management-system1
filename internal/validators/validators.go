// Package validators provides input validation helpers for all Go handlers.
package validators

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ValidationError holds a map of field → error message.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	var msgs []string
	for field, msg := range e.Fields {
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}

// Validator accumulates validation errors for a request.
type Validator struct {
	errors map[string]string
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{errors: make(map[string]string)}
}

// Required checks that a string field is not empty.
func (v *Validator) Required(field, value, label string) {
	if strings.TrimSpace(value) == "" {
		v.errors[field] = fmt.Sprintf("%s is required", label)
	}
}

// MinLength checks that a string meets a minimum length.
func (v *Validator) MinLength(field, value string, min int, label string) {
	if len(strings.TrimSpace(value)) < min {
		v.errors[field] = fmt.Sprintf("%s must be at least %d characters", label, min)
	}
}

// MaxLength checks that a string does not exceed a maximum length.
func (v *Validator) MaxLength(field, value string, max int, label string) {
	if len(value) > max {
		v.errors[field] = fmt.Sprintf("%s must not exceed %d characters", label, max)
	}
}

// Email validates an email address format.
func (v *Validator) Email(field, value string) {
	if _, err := mail.ParseAddress(value); err != nil {
		v.errors[field] = "Invalid email address"
	}
}

// Password validates password strength.
// Requires: min 8 chars, at least one uppercase, one lowercase, one digit.
func (v *Validator) Password(field, value string) {
	if len(value) < 8 {
		v.errors[field] = "Password must be at least 8 characters"
		return
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range value {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		v.errors[field] = "Password must contain at least one uppercase letter, one lowercase letter, and one number"
	}
}

// PasswordMatch checks that two password fields match.
func (v *Validator) PasswordMatch(field, password, confirm string) {
	if password != confirm {
		v.errors[field] = "Passwords do not match"
	}
}

// Phone validates a phone number (basic — allows +, digits, spaces, dashes).
func (v *Validator) Phone(field, value string) {
	re := regexp.MustCompile(`^\+?[\d\s\-]{7,20}$`)
	if !re.MatchString(strings.TrimSpace(value)) {
		v.errors[field] = "Invalid phone number"
	}
}

// Date validates a date string in YYYY-MM-DD format.
func (v *Validator) Date(field, value string) {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		v.errors[field] = "Invalid date format (expected YYYY-MM-DD)"
	}
}

// InRange checks that an integer is within [min, max].
func (v *Validator) InRange(field string, value, min, max int, label string) {
	if value < min || value > max {
		v.errors[field] = fmt.Sprintf("%s must be between %d and %d", label, min, max)
	}
}

// ScoreNotExceed checks that a score does not exceed the maximum.
func (v *Validator) ScoreNotExceed(field string, score, max float64, componentName string) {
	if score > max {
		v.errors[field] = fmt.Sprintf("%s score (%.1f) exceeds maximum (%.1f)", componentName, score, max)
	}
	if score < 0 {
		v.errors[field] = fmt.Sprintf("%s score cannot be negative", componentName)
	}
}

// ScoreTotalEquals100 checks that CA total + exam marks = 100.
func (v *Validator) ScoreTotalEquals100(field string, caTotal, examMarks int) {
	if caTotal+examMarks != 100 {
		v.errors[field] = fmt.Sprintf("CA total (%d) + Exam marks (%d) must equal 100", caTotal, examMarks)
	}
}

// CloudinaryURL validates that a URL is from Cloudinary.
func (v *Validator) CloudinaryURL(field, value string) {
	if value == "" {
		return // optional fields are allowed to be empty
	}
	if !strings.HasPrefix(value, "https://res.cloudinary.com/") {
		v.errors[field] = "Invalid file URL — must be a Cloudinary URL"
	}
}

// Enum checks that a value is one of the allowed options.
func (v *Validator) Enum(field, value string, allowed []string, label string) {
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	v.errors[field] = fmt.Sprintf("%s must be one of: %s", label, strings.Join(allowed, ", "))
}

// Custom adds a custom error message for a field.
func (v *Validator) Custom(field, message string) {
	v.errors[field] = message
}

// Errors returns the validation error map.
func (v *Validator) Errors() map[string]string {
	return v.errors
}

// HasErrors returns true if there are any validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// ToError returns a ValidationError if there are errors, or nil.
func (v *Validator) ToError() error {
	if !v.HasErrors() {
		return nil
	}
	return &ValidationError{Fields: v.errors}
}
