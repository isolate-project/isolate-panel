package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/isolate-project/isolate-panel/internal/auth"
)

func SubscriptionSignatureValidator(signer *auth.SubscriptionSigner) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Params("token")
		if token == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Missing subscription token")
		}

		rawSig := c.Query("sig")
		rawExp := c.Query("exp")

		if rawSig == "" || rawExp == "" {
			return c.Status(fiber.StatusForbidden).SendString("Subscription URL requires signature")
		}

		sig, exp, err := auth.ParseSubscriptionQuery(rawSig, rawExp)
		if err != nil {
			return c.Status(fiber.StatusForbidden).SendString("Invalid subscription URL parameters")
		}

		if !signer.Verify(token, sig, exp) {
			return c.Status(fiber.StatusForbidden).SendString("Invalid or expired subscription URL")
		}

		return c.Next()
	}
}
