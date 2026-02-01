package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv                string `mapstructure:"APP_ENV"`
	RabbitMQURL           string `mapstructure:"RABBITMQ_URL"`
	ElasticsearchURL      string `mapstructure:"ELASTICSEARCH_URL"`
	ElasticsearchIndex    string `mapstructure:"ELASTICSEARCH_INDEX"`
	UserServiceURL        string `mapstructure:"USER_SERVICE_URL"`
	ContentServiceURL     string `mapstructure:"CONTENT_SERVICE_URL"`
	InternalServiceSecret string `mapstructure:"INTERNAL_SERVICE_SECRET"`
	DatabaseURL           string `mapstructure:"USER_DATABASE_URL"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("ELASTICSEARCH_INDEX", "showcases")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("⚠️  No .env file found, relying on System Env Vars")
	}

	viper.BindEnv("RABBITMQ_URL")
	viper.BindEnv("ELASTICSEARCH_URL")
	viper.BindEnv("USER_DATABASE_URL")
	viper.BindEnv("USER_SERVICE_URL")
	viper.BindEnv("CONTENT_SERVICE_URL")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.RabbitMQURL == "" {
		cfg.RabbitMQURL = viper.GetString("RABBITMQ_URL")
	}

	fmt.Printf("🔍 DEBUG CONFIG LOADED:\n")
	fmt.Printf("   RabbitMQ: %s\n", cfg.RabbitMQURL)
	fmt.Printf("   Elastic:  %s\n", cfg.ElasticsearchURL)
	fmt.Printf("   UserService: %s\n", cfg.UserServiceURL)
	fmt.Printf("   ContentService: %s\n", cfg.ContentServiceURL)

	return &cfg, nil
}
