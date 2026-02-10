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
	SearchShowcases(ctx context.Context, query string, limit int, cursor []interface{}, filter SearchFilter) ([]entity.Showcase, []interface{}, error)
	GetTrendingShowcases(ctx context.Context, limit int, offset int) ([]entity.Showcase, error)
}

type SearchFilter struct {
	CategoryIDs []uuid.UUID
	SortBy      string // "latest", "popular", "relevance"
}

type searchRepository struct {
	client    *elasticsearch8.Client
	indexName string
	log       *logrus.Logger
}

func NewSearchRepository(client *elasticsearch8.Client, cfg *config.Config, log *logrus.Logger) SearchRepository {
	indexName := cfg.ElasticsearchIndex
	return &searchRepository{
		client:    client,
		indexName: indexName,
		log:       log,
	}
}

func (r *searchRepository) SearchShowcases(ctx context.Context, query string, limit int, cursor []interface{}, filter SearchFilter) ([]entity.Showcase, []interface{}, error) {
	var buf bytes.Buffer

	// Build Query
	var queryMap map[string]interface{}
	var boolQuery map[string]interface{}

	if query == "" {
		boolQuery = map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{"match_all": map[string]interface{}{}},
			},
		}
	} else {
		boolQuery = map[string]interface{}{
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
		}
	}

	// Apply Filters (Category)
	if len(filter.CategoryIDs) > 0 {
		terms := make([]interface{}, len(filter.CategoryIDs))
		for i, id := range filter.CategoryIDs {
			terms[i] = id.String()
		}

		filterClause := map[string]interface{}{
			"terms": map[string]interface{}{
				"category_id": terms,
			},
		}

		// Append to filter context (efficient, cachable)
		if existingFilter, ok := boolQuery["filter"].([]interface{}); ok {
			boolQuery["filter"] = append(existingFilter, filterClause)
		} else {
			boolQuery["filter"] = []interface{}{filterClause}
		}
	}

	queryMap = map[string]interface{}{
		"bool": boolQuery,
	}

	// Determine Sort
	var sortClause []map[string]interface{}
	switch filter.SortBy {
	case "latest":
		sortClause = []map[string]interface{}{
			{"created_at": "desc"},
			{"id": "asc"},
		}
	case "popular":
		sortClause = []map[string]interface{}{
			{"like_count": "desc"},
			{"id": "asc"},
		}
	default: // "relevance" or default
		if query == "" {
			// If no query but sort is relevance, fallback to latest
			sortClause = []map[string]interface{}{
				{"created_at": "desc"},
				{"id": "asc"},
			}
		} else {
			sortClause = []map[string]interface{}{
				{"_score": "desc"},
				{"id": "asc"},
			}
		}
	}

	searchSource := map[string]interface{}{
		"query": queryMap,
		"size":  limit,
		"sort":  sortClause,
	}

	// Add Search After if cursor exists
	if len(cursor) > 0 {
		searchSource["search_after"] = cursor
	}

	if err := json.NewEncoder(&buf).Encode(searchSource); err != nil {
		return nil, nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	// Perform Search
	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform search request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, nil, fmt.Errorf("error parsing error response: %w", err)
		}
		return nil, nil, fmt.Errorf("search error: %s", e["error"]) // Simplify error rep
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
				Source esShowcase    `json:"_source"`
				Sort   []interface{} `json:"sort"` // Capture sort values for next cursor
			} `json:"hits"`
		} `json:"hits"`
	}

	var rResponse esResponse
	if err := json.NewDecoder(res.Body).Decode(&rResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	showcases := make([]entity.Showcase, 0, len(rResponse.Hits.Hits))
	var lastSortValues []interface{}

	for _, hit := range rResponse.Hits.Hits {
		src := hit.Source
		lastSortValues = hit.Sort // Keep updating to the last one

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

	return showcases, lastSortValues, nil
}

func (r *searchRepository) GetTrendingShowcases(ctx context.Context, limit int, offset int) ([]entity.Showcase, error) {
	var buf bytes.Buffer

	// Script Score Query
	script := `
		double views = (doc['view_count'].size() > 0) ? doc['view_count'].value : 0;
		double likes = (doc['like_count'].size() > 0) ? doc['like_count'].value : 0;
		double comments = (doc['comment_count'].size() > 0) ? doc['comment_count'].value : 0;
		double collabs = (doc['collaborators.id'].size() > 0) ? doc['collaborators.id'].size() : 0;
		return (views * 1) + (likes * 10) + (comments * 30) + (collabs * 50);
	`

	queryMap := map[string]interface{}{
		"function_score": map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": []map[string]interface{}{
						{
							"range": map[string]interface{}{
								"created_at": map[string]interface{}{
									"gte": "now-7d/d",
									"lt":  "now/d+1d", // Up to end of today
								},
							},
						},
					},
				},
			},
			"script_score": map[string]interface{}{
				"script": map[string]interface{}{
					"source": script,
				},
			},
			"boost_mode": "replace", // Use script score as the _score
		},
	}

	searchSource := map[string]interface{}{
		"query": queryMap,
		"from":  offset,
		"size":  limit,
		"sort": []map[string]interface{}{
			{"_score": "desc"},
			{"id": "asc"},
		},
	}

	if err := json.NewEncoder(&buf).Encode(searchSource); err != nil {
		return nil, fmt.Errorf("failed to encode trending query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to perform trending search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("trending search error: %s", res.String())
	}

	// Reusing parsing logic (Inline for now to avoid huge refactor, but kept clean)
	// TODO: Refactor common parsing logic
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
		entity.Showcase                  // Embed base fields
		Owner           esOwner          `json:"owner"`
		Collaborators   []esCollaborator `json:"collaborators"`
	}

	type esResponse struct {
		Hits struct {
			Hits []struct {
				Source esShowcase `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	var rResponse esResponse
	if err := json.NewDecoder(res.Body).Decode(&rResponse); err != nil {
		return nil, fmt.Errorf("failed to decode trending response: %w", err)
	}

	showcases := make([]entity.Showcase, 0, len(rResponse.Hits.Hits))
	for _, hit := range rResponse.Hits.Hits {
		src := hit.Source
		finalShowcase := src.Showcase

		var enriched []entity.EnrichedCollaborator

		// Add Owner
		if src.Owner.ID != "" {
			ownerID, _ := uuid.Parse(src.Owner.ID)
			ownerUserIDStr := src.Owner.UserID
			if ownerUserIDStr == "" {
				ownerUserIDStr = src.Owner.ID
			}
			ownerUserID, _ := uuid.Parse(ownerUserIDStr)

			enriched = append(enriched, entity.EnrichedCollaborator{
				ID:     ownerID,
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

		// Add Collaborators
		for _, c := range src.Collaborators {
			cID, _ := uuid.Parse(c.ID)
			cUserIDStr := c.UserID
			if cUserIDStr == "" {
				cUserIDStr = c.ID
			}
			uID, _ := uuid.Parse(cUserIDStr)

			enriched = append(enriched, entity.EnrichedCollaborator{
				ID:     cID,
				Role:   "COLLABORATOR",
				Status: "ACCEPTED",
				User: &entity.User{
					ID:        uID,
					Username:  c.Username,
					Name:      c.FullName,
					AvatarURL: c.AvatarURL,
				},
			})
		}

		finalShowcase.EnrichedCollaborators = enriched
		showcases = append(showcases, finalShowcase)
	}

	return showcases, nil
}
