package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/sirupsen/logrus"
)

// NewRedisClient initializes and returns a Redis client
func NewRedisClient(cfg *config.Config, log *logrus.Logger) (*redis.Client, error) {
	// Parse URL (e.g. redis://user:password@localhost:6379/0 or just localhost:6379)
	var opts *redis.Options
	var err error

	if strings.HasPrefix(cfg.RedisURL, "redis://") || strings.HasPrefix(cfg.RedisURL, "rediss://") {
		opts, err = redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid redis url: %w", err)
		}
	} else {
		opts = &redis.Options{
			Addr: cfg.RedisURL,
		}
	}

	client := redis.NewClient(opts)

	// Ping to verify connection
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Info("Connected to Redis successfully")
	return client, nil
}
