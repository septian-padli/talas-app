package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/pkg/utils"
)

type AuthMiddleware struct {
	cfg *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (m *AuthMiddleware) Protect(c *fiber.Ctx) error {
	// 1. Get Token from Cookie "access_token"
	// User service uses lowercase "accessToken" in JS, need to verify cookie name from response
	// Let's support both "access_token" and "accessToken" or Authorization header (Bearer)
	
	// Priority 1: Cookie
	tokenString := c.Cookies("access_token")
	if tokenString == "" {
		// Fallback Cookie Name (camelCase)
		tokenString = c.Cookies("accessToken")
	}

	// Priority 2: Header (Optional, for API testing)
	if tokenString == "" {
		authHeader := c.Get("Authorization")
		if len(authHeader) > 7 && strings.ToUpper(authHeader[:7]) == "BEARER " {
			tokenString = authHeader[7:]
		}
	}

	if tokenString == "" {
		return utils.ErrorResponse(c, 401, "Unauthorized: Harap login terlebih dahulu", nil)
	}

	// 2. Verify Token
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// Validate Algo
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(m.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return utils.ErrorResponse(c, 401, "Unauthorized: Token tidak valid atau expired", nil)
	}

	// 3. Extract Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return utils.ErrorResponse(c, 401, "Unauthorized: Invalid token claims", nil)
	}

	// 4. Set to Context
	// User Service claims: "id", "email"
	userID := claims["id"]
	if userID == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized: User ID not found in token", nil)
	}

	c.Locals("user_id", userID)
	
	return c.Next()
}
