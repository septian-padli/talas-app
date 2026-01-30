package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

func NewLogger() *logrus.Logger {
	log := logrus.New()
	
	// Output to stdout as requested
	log.SetOutput(os.Stdout)
	
	// Use JSON Formatter
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00", // ISO8601
	})

	// Set Level from Env
	levelStr := os.Getenv("LOG_LEVEL")
	level, err := logrus.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		level = logrus.InfoLevel // Default
	}
	log.SetLevel(level)

	return log
}
