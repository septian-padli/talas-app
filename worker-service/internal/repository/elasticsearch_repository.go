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
	CreateIndexIfNotExists(ctx context.Context) error
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
					"id":         collaborator.ID,
					"user_id":    collaborator.UserID,
					"username":   collaborator.Username,
					"full_name":  collaborator.FullName,
					"avatar_url": collaborator.AvatarURL,
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
			ctx._source.collaborators.removeIf(item -> item.user_id == params.user_id);
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
} // CreateIndexIfNotExists creates the index with explicit mapping if it doesn't exist
func (r *elasticsearchRepository) CreateIndexIfNotExists(ctx context.Context) error {
	resOrErr, err := r.client.Indices.Exists([]string{r.indexName}, r.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer resOrErr.Body.Close()

	if resOrErr.StatusCode == 200 {
		return nil
	}

	mapping := `
	{
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"slug": { "type": "keyword" },
				"title": { "type": "text" },
				"content": { "type": "text" },
				"category_id": { "type": "keyword" },
				"tags": { "type": "keyword" },
				"created_at": { "type": "date" },
				"updated_at": { "type": "date" },
				"like_count": { "type": "integer" },
				"view_count": { "type": "integer" },
				"owner": {
					"properties": {
						"id": { "type": "keyword" },
						"user_id": { "type": "keyword" },
						"username": { "type": "text" },
						"full_name": { "type": "text" },
						"avatar_url": { "type": "keyword" }
					}
				},
				"collaborators": {
					"properties": {
						"id": { "type": "keyword" },
						"user_id": { "type": "keyword" },
						"username": { "type": "text" },
						"full_name": { "type": "text" },
						"avatar_url": { "type": "keyword" }
					}
				}
			}
		}
	}`

	res, err := r.client.Indices.Create(
		r.indexName,
		r.client.Indices.Create.WithBody(bytes.NewReader([]byte(mapping))),
		r.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES create index error: %s", res.String())
	}

	r.log.Infof("Created index %s with explicit mapping", r.indexName)
	return nil
}
