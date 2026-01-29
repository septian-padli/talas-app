package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/septianpadli/talas/content-service/tests/helper"
	"github.com/stretchr/testify/assert"
)

// --- 1. SETUP APP DUMMY ---
// Fungsi ini meniru apa yang terjadi di main.go, tapi versi minimalis
func setupTestApp() *fiber.App {
	app := fiber.New()

	// Set ENV variable agar sinkron antara Helper dan Middleware
	os.Setenv("JWT_SECRET", "secret-untuk-test")

	// Middleware Auth Sederhana (Simulasi Middleware Asli Kamu)
	// Tugasnya: Validasi Token -> Set c.Locals("user_id")
	authMiddleware := func(c *fiber.Ctx) error {
		cookie := c.Cookies("access_token")
		if cookie == "" {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}

		token, err := jwt.Parse(cookie, func(t *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid Token"})
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Locals("user_id", claims["id"]) // Menggunakan "id" sesuai middleware asli
		return c.Next()
	}

	// Group Routes
	api := app.Group("/api")
	protected := api.Group("/", authMiddleware) // Hilangkan /v1 agar sama dengan provider.go

	// Endpoint yang mau dites
	protected.Get("/test", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Authenticated!",
			"user_id": userID,
		})
	})

	return app
}

// --- 2. TEST CASE UTAMA ---
func TestGetProtectedToken(t *testing.T) {
	// Setup Fiber App
	app := setupTestApp()

	// Data Dummy
	targetUserID := "550e8400-e29b-41d4-a716-446655440000"

	// Step A: Bikin Token Valid pakai Helper
	token, err := helper.GenerateTestToken(targetUserID)
	assert.NoError(t, err)

	// Step B: Bikin Request HTTP
	req := httptest.NewRequest("GET", "/api/test", nil)
	
	// Step C: Masukkan Token ke Cookie (Sesuai kontrak API)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: token,
	})

	// Step D: Jalankan Request via Fiber Test Method
	// -1 artinya disable timeout (tunggu sampai selesai)
	resp, err := app.Test(req, -1)

	// --- 3. ASSERTIONS (Testify Magic) ---
	
	// Cek Error HTTP
	assert.NoError(t, err)
	
	// Cek Status Code harus 200 OK
	assert.Equal(t, 200, resp.StatusCode)

	// Cek Response Body
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	// Pastikan user_id yang dikirim token SAMA dengan yang diterima response
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, targetUserID, body["user_id"])
}

// --- 3. TEST CASE NEGATIF (Opsional tapi Bagus) ---
func TestGetProtectedToken_Unauthorized(t *testing.T) {
	app := setupTestApp()

	// Request TANPA Cookie
	req := httptest.NewRequest("GET", "/api/test", nil)
	
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode) // Harusnya Unauthorized
}