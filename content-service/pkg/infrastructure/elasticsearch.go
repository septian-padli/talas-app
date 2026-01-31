package infrastructure

import (
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/septianpadli/talas/content-service/internal/config"
)

// NewElasticsearchClient initializes a new Elasticsearch v8 client
func NewElasticsearchClient(cfg *config.Config) (*elasticsearch8.Client, error) {
	client, err := elasticsearch8.NewClient(elasticsearch8.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	// Ping to verify connection
	info, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer info.Body.Close()

	return client, nil
}
