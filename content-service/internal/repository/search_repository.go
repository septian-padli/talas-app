package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/sirupsen/logrus"
)

type SearchRepository interface {
	SearchShowcases(ctx context.Context, query string, page int, limit int) ([]entity.Showcase, int64, error)
}

type searchRepository struct {
	client    *elasticsearch8.Client
	indexName string
	log       *logrus.Logger
}

func NewSearchRepository(client *elasticsearch8.Client, cfg *config.Config, log *logrus.Logger) SearchRepository {
	// Assuming index name is hardcoded or from config, but typically "showcases" based on worker service
	indexName := "showcases"
	return &searchRepository{
		client:    client,
		indexName: indexName,
		log:       log,
	}
}

func (r *searchRepository) SearchShowcases(ctx context.Context, query string, page int, limit int) ([]entity.Showcase, int64, error) {
	from := (page - 1) * limit
	var buf bytes.Buffer

	// Build Query
	queryMap := map[string]interface{}{}

	if query == "" {
		queryMap = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	} else {
		queryMap = map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					// 1. Metadata Fields (High Priority, Fuzzy Enabled)
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query": query,
							"fields": []string{
								"title^3",
								"owner.username^2",
								"owner.full_name^2",
								"slug",
								"collaborators.username",
								"collaborators.full_name",
								"tags^2",
							},
							"fuzziness": "AUTO",
						},
					},
					// 2. Content Field (Lower Priority, Strict/No Fuzzy for Performance)
					map[string]interface{}{
						"match": map[string]interface{}{
							"content": map[string]interface{}{
								"query":     query,
								"boost":     0.5,
								"fuzziness": "0", // Disable fuzzy for long text
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	searchSource := map[string]interface{}{
		"query": queryMap,
		"from":  from,
		"size":  limit,
	}

	if err := json.NewEncoder(&buf).Encode(searchSource); err != nil {
		return nil, 0, fmt.Errorf("failed to encode search query: %w", err)
	}

	// Perform Search
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to perform search request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, 0, fmt.Errorf("error parsing error response: %w", err)
		}
		return nil, 0, fmt.Errorf("search error: %s", e["error"]) // Simplify error rep
	}

	// Parse Response
	// 1. Define Intermediate Struct matching ES Document
	type esCollaborator struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		FullName  string `json:"full_name"`
		AvatarURL string `json:"avatar_url"`
	}

	type esOwner struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		FullName  string `json:"full_name"`
		AvatarURL string `json:"avatar_url"`
	}

	type esShowcase struct {
		entity.Showcase                  // Embed base fields for flat mapping (Title, Slug, etc)
		Owner           esOwner          `json:"owner"`
		Collaborators   []esCollaborator `json:"collaborators"`
	}

	type esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source esShowcase `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	var rResponse esResponse
	if err := json.NewDecoder(res.Body).Decode(&rResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to decode search response: %w", err)
	}

	showcases := make([]entity.Showcase, 0, len(rResponse.Hits.Hits))
	for _, hit := range rResponse.Hits.Hits {
		src := hit.Source

		// 2. Map to Domain Entity
		finalShowcase := src.Showcase // Copy embedded fields

		// 3. Construct Enriched Collaborators
		var enriched []entity.EnrichedCollaborator

		// Add Owner as OWNER
		if src.Owner.ID != "" {
			ownerID, _ := uuid.Parse(src.Owner.ID)

			// Owner User ID Fallback
			ownerUserIDStr := src.Owner.UserID
			if ownerUserIDStr == "" {
				ownerUserIDStr = src.Owner.ID
			}
			ownerUserID, _ := uuid.Parse(ownerUserIDStr)

			enriched = append(enriched, entity.EnrichedCollaborator{
				ID:     ownerID, // Use User ID as fallback for Collab ID in search for OWNER
				Role:   "OWNER",
				Status: "ACCEPTED",
				User: &entity.User{
					ID:        ownerUserID,
					Username:  src.Owner.Username,
					Name:      src.Owner.FullName,
					AvatarURL: src.Owner.AvatarURL,
				},
			})
		}

		// Add Other Collaborators
		for _, c := range src.Collaborators {
			cID, _ := uuid.Parse(c.ID)

			// Collaborator User ID Fallback
			cUserIDStr := c.UserID
			if cUserIDStr == "" {
				cUserIDStr = c.ID
			}
			uID, _ := uuid.Parse(cUserIDStr)

			enriched = append(enriched, entity.EnrichedCollaborator{
				ID:     cID,            // Collaboration ID (Relationship)
				Role:   "COLLABORATOR", // Default role
				Status: "ACCEPTED",     // Assumed accepted
				User: &entity.User{
					ID:        uID, // User ID
					Username:  c.Username,
					Name:      c.FullName,
					AvatarURL: c.AvatarURL,
				},
			})
		}

		finalShowcase.EnrichedCollaborators = enriched
		showcases = append(showcases, finalShowcase)
	}

	return showcases, rResponse.Hits.Total.Value, nil
}
