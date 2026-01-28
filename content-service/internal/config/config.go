package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Port        string
	DatabaseURL string
	// DB Specifics
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string
	DBSSLMode  string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		// Try loading from root if in cmd/api
		if err := godotenv.Load("../../.env"); err != nil {
			logrus.Warn("No .env file found, using system environment variables")
		}
	}

	return &Config{
		Port:       getEnv("PORT", "3002"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "talas_content"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
