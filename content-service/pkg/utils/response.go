package utils

import "github.com/gofiber/fiber/v2"

type ApiResponse struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func SuccessResponse(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(code).JSON(ApiResponse{
		Code:    code,
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *fiber.Ctx, code int, message string, errors interface{}) error {
	return c.Status(code).JSON(ApiResponse{
		Code:    code,
		Success: false,
		Message: message,
		Errors:  errors,
	})
}
