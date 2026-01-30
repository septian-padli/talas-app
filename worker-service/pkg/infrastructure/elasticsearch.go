package infrastructure

import (
	"crypto/tls"
	"fmt"
	"net/http"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/septian/worker-service/internal/config"
	"github.com/sirupsen/logrus"
)

func NewElasticsearchClient(cfg *config.Config, log *logrus.Logger) (*elasticsearch8.Client, error) {
	esCfg := elasticsearch8.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	}

	// Skip SSL verification in development
	if cfg.AppEnv == "development" {
		esCfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client, err := elasticsearch8.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ES client: %w", err)
	}

	// Verify connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to ping ES: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES returned error: %s", res.String())
	}

	log.Info("Successfully connected to Elasticsearch")
	return client, nil
}
