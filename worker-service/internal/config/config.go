package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv             string `mapstructure:"APP_ENV"`
	RabbitMQURL        string `mapstructure:"RABBITMQ_URL"`
	ElasticsearchURL   string `mapstructure:"ELASTICSEARCH_URL"`
	ElasticsearchIndex string `mapstructure:"ELASTICSEARCH_INDEX"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("ELASTICSEARCH_INDEX", "showcases")

	if err := viper.ReadInConfig(); err != nil {
		// Log warning but continue (env vars might be set)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
