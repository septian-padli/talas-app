package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// NewRequestLogger returns a fiber middleware that logs HTTP requests
func NewRequestLogger(log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process Request
		err := c.Next()

		// Calculate Latency
		latency := time.Since(start)

		// Get Request ID from Locals (set by requestid middleware)
		reqID, _ := c.Locals("requestid").(string)
		
		// Log Fields
		fields := logrus.Fields{
			"request_id": reqID,
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"latency":    latency.String(),
			"ip":         c.IP(),
		}

		if err != nil {
			fields["error"] = err.Error()
		}

		// Log Entry
		log.WithFields(fields).Info("HTTP Request")

		return err
	}
}
