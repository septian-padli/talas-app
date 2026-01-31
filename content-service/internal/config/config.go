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

	// Cloudinary
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string

	// Auth
	JWTSecret string

	// Services
	UserServiceURL        string
	InternalServiceSecret string
	RabbitMQURL           string
	ElasticsearchURL      string
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

		// Cloudinary
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),

		// Auth
		JWTSecret: getEnv("JWT_SECRET", "supersecretkey"),

		// Services
		UserServiceURL:        getEnv("USER_SERVICE_URL", "http://localhost:3001"),
		InternalServiceSecret: getEnv("INTERNAL_SERVICE_SECRET", ""),
		RabbitMQURL:           getEnv("RABBITMQ_URL", "amqp://user:password@localhost:5672"),
		ElasticsearchURL:      getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
