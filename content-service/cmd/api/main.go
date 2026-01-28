package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize App via Wire (Dependency Injection)
	// Config, Logger, Database, etc are handled inside here
	app, err := InitializeApp()
	if err != nil {
		// Fallback logger since our wire logger might not be ready if init fails
		logrus.Fatalf("Failed to initialize app: %v", err)
	}

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	logrus.Infof("Server running on port %s", port)
	logrus.Fatal(app.Listen(":" + port))
}