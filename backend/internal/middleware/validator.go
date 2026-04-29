package middleware

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/isolate-project/isolate-panel/internal/validators"
)

var validate = validator.New()

func SetupValidation() {
	validators.RegisterCustomValidators(validate)
}

// BindAndValidate binds the request JSON body into T and validates it using
// the struct's `validate` tags. Returns a fiber.Error on bind or validation
// failure so that the global ErrorHandler produces a consistent 400 response.
func BindAndValidate[T any](c fiber.Ctx) (T, error) {
	var req T
	if err := c.Bind().JSON(&req); err != nil {
		return req, fiber.NewError(fiber.StatusBadRequest, "Invalid JSON: "+err.Error())
	}
	if err := validate.Struct(req); err != nil {
		return req, fiber.NewError(fiber.StatusBadRequest, formatValidationErrors(err))
	}
	return req, nil
}

func formatValidationErrors(err error) string {
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		msgs := make([]string, 0, len(verr))
		for _, fe := range verr {
			msgs = append(msgs, fe.Field()+": "+fe.Tag())
		}
		return strings.Join(msgs, "; ")
	}
	return err.Error()
}
