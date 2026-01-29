package helper

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateTestToken membuat token valid dengan User ID tertentu
func GenerateTestToken(userID string) (string, error) {
	// 1. Definisikan Claims (Data user)
	claims := jwt.MapClaims{
		"id":       userID,
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour * 1).Unix(), // Expired 1 jam
	}

	// 2. Buat Token Object
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. Ambil Secret Key (Harus sama dengan yang dipakai di App/Middleware)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "secret-untuk-test" // Default fallback biar test gak panic
	}

	// 4. Tanda tangani token
	return token.SignedString([]byte(secret))
}