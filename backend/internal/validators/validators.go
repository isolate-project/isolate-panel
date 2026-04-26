package validators

import (
	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation functions with the validator instance
func RegisterCustomValidators(v *validator.Validate) {
	v.RegisterValidation("alphanum_special", validatePasswordComplexity)
}

// validatePasswordComplexity validates that a password contains:
// - At least 12 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one digit
// - At least one special character
func validatePasswordComplexity(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 12 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}