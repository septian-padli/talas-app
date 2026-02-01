package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/septianpadli/talas/content-service/internal/config"
)

// InternalAuthMiddleware validates the x-service-secret header
func InternalAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := c.Get("x-service-secret")

		// Check if header is missing
		if secret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Missing x-service-secret header",
			})
		}

		// Validate secret
		if secret != cfg.InternalServiceSecret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Invalid internal service secret",
			})
		}

		// Secret is valid, proceed
		return c.Next()
	}
}
