package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/septianpadli/talas/content-service/internal/config"
)

// InternalAuthMiddleware validates the x-service-secret header
func InternalAuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		secret := c.Get("x-service-secret")
		fmt.Println("Received x-service-secret:", secret)
		fmt.Println("Expected internal service secret:", cfg.InternalServiceSecret)

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
