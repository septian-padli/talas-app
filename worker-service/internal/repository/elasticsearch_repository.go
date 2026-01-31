package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/septian/worker-service/internal/config"
	"github.com/septian/worker-service/internal/domain"
	"github.com/sirupsen/logrus"
)

// ElasticsearchRepository defines the interface for ES operations
type ElasticsearchRepository interface {
	IndexShowcase(ctx context.Context, showcase *domain.Showcase) error
	DeleteShowcase(ctx context.Context, id string) error
	AddCollaborator(ctx context.Context, showcaseID string, collaborator domain.Collaborator) error
	RemoveCollaborator(ctx context.Context, showcaseID string, userID string) error
}

// elasticsearchRepository implements ElasticsearchRepository
type elasticsearchRepository struct {
	client    *elasticsearch8.Client
	indexName string
	log       *logrus.Logger
}

// NewElasticsearchRepository creates a new ES repository instance
func NewElasticsearchRepository(client *elasticsearch8.Client, cfg *config.Config, log *logrus.Logger) ElasticsearchRepository {
	return &elasticsearchRepository{
		client:    client,
		indexName: cfg.ElasticsearchIndex,
		log:       log,
	}
}

// IndexShowcase indexes or updates a showcase document in Elasticsearch
func (r *elasticsearchRepository) IndexShowcase(ctx context.Context, showcase *domain.Showcase) error {
	data, err := json.Marshal(showcase)
	if err != nil {
		return fmt.Errorf("failed to marshal showcase: %w", err)
	}

	res, err := r.client.Index(
		r.indexName,
		bytes.NewReader(data),
		r.client.Index.WithDocumentID(showcase.ID.String()),
		r.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to index showcase: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES index error: %s", res.String())
	}

	r.log.Infof("Indexed showcase %s into %s", showcase.ID, r.indexName)
	return nil
}

// DeleteShowcase removes a showcase document from Elasticsearch
func (r *elasticsearchRepository) DeleteShowcase(ctx context.Context, id string) error {
	res, err := r.client.Delete(
		r.indexName,
		id,
		r.client.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to delete showcase: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("ES delete error: %s", res.String())
	}

	r.log.Infof("Deleted showcase %s from %s", id, r.indexName)
	return nil
}

// AddCollaborator adds a collaborator to the showcase document using a script
// AddCollaborator adds a collaborator to the showcase document using a script
func (r *elasticsearchRepository) AddCollaborator(ctx context.Context, showcaseID string, collaborator domain.Collaborator) error {
	script := `
		if (ctx._source.collaborators == null) { ctx._source.collaborators = new ArrayList(); }
		boolean exists = false;
		for (item in ctx._source.collaborators) { if (item.id == params.collab.id) { exists = true; } }
		if (!exists) { ctx._source.collaborators.add(params.collab); }
	`

	requestBody := map[string]interface{}{
		"script": map[string]interface{}{
			"source": script,
			"lang":   "painless",
			"params": map[string]interface{}{
				"collab": map[string]interface{}{
					"id":       collaborator.ID,
					"username": collaborator.Username,
				},
			},
		},
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal add collab script: %w", err)
	}

	res, err := r.client.Update(
		r.indexName,
		showcaseID,
		bytes.NewReader(data),
		r.client.Update.WithContext(ctx),
	)

	if err != nil {
		return fmt.Errorf("failed to add collaborator: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES update error: %s", res.String())
	}

	r.log.Infof("Added collaborator %s to showcase %s", collaborator.ID, showcaseID)
	return nil
}

// RemoveCollaborator removes a collaborator from the showcase document using a script
// RemoveCollaborator removes a collaborator from the showcase document using a script
func (r *elasticsearchRepository) RemoveCollaborator(ctx context.Context, showcaseID string, userID string) error {
	script := `
		if (ctx._source.collaborators != null) {
			ctx._source.collaborators.removeIf(item -> item.id == params.user_id);
		}
	`

	requestBody := map[string]interface{}{
		"script": map[string]interface{}{
			"source": script,
			"lang":   "painless",
			"params": map[string]interface{}{
				"user_id": userID,
			},
		},
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal remove collab script: %w", err)
	}

	res, err := r.client.Update(
		r.indexName,
		showcaseID,
		bytes.NewReader(data),
		r.client.Update.WithContext(ctx),
	)

	if err != nil {
		return fmt.Errorf("failed to remove collaborator: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES update error: %s", res.String())
	}

	r.log.Infof("Removed collaborator %s from showcase %s", userID, showcaseID)
	return nil
}
