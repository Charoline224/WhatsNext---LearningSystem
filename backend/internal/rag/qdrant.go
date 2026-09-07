package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxQdrantResponseBytes = 8 << 20

type ChunkPoint struct {
	ID              string
	Vector          []float32
	UserID          string
	LearningSpaceID string
	MaterialID      string
	ChunkID         string
	ChunkIndex      int
}

type SearchResult struct {
	PointID string
	Score   float32
	Payload map[string]any
}

type Qdrant struct {
	baseURL    string
	apiKey     string
	collection string
	dimensions int
	client     *http.Client
}

func NewQdrant(baseURL, apiKey, collection string, dimensions int, timeout time.Duration) (*Qdrant, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if _, err := url.ParseRequestURI(baseURL); err != nil || baseURL == "" {
		return nil, fmt.Errorf("invalid qdrant URL")
	}
	if strings.TrimSpace(collection) == "" || dimensions <= 0 || timeout <= 0 {
		return nil, fmt.Errorf("qdrant collection, dimensions, and timeout must be configured")
	}
	return &Qdrant{baseURL: baseURL, apiKey: apiKey, collection: collection, dimensions: dimensions, client: &http.Client{Timeout: timeout}}, nil
}

func (q *Qdrant) EnsureCollection(ctx context.Context) error {
	status, response, err := q.do(ctx, http.MethodGet, "/collections/"+url.PathEscape(q.collection), nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return q.ensurePayloadIndexes(ctx)
	}
	if status != http.StatusNotFound {
		return qdrantStatusError("inspect collection", status, response)
	}
	body := map[string]any{"vectors": map[string]any{"size": q.dimensions, "distance": "Cosine"}}
	status, response, err = q.do(ctx, http.MethodPut, "/collections/"+url.PathEscape(q.collection), body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return qdrantStatusError("create collection", status, response)
	}
	return q.ensurePayloadIndexes(ctx)
}

func (q *Qdrant) ensurePayloadIndexes(ctx context.Context) error {
	for _, field := range []string{"user_id", "learning_space_id", "material_id", "chunk_id"} {
		body := map[string]any{"field_name": field, "field_schema": "keyword"}
		status, response, err := q.do(ctx, http.MethodPut, "/collections/"+url.PathEscape(q.collection)+"/index?wait=true", body)
		if err != nil {
			return err
		}
		if status < 200 || status >= 300 {
			return qdrantStatusError("create payload index "+field, status, response)
		}
	}
	return nil
}

func (q *Qdrant) Upsert(ctx context.Context, points []ChunkPoint) error {
	if len(points) == 0 {
		return nil
	}
	encoded := make([]map[string]any, len(points))
	for i, point := range points {
		if point.ID == "" || point.UserID == "" || point.LearningSpaceID == "" || point.MaterialID == "" || point.ChunkID == "" {
			return fmt.Errorf("qdrant point identifiers must not be empty")
		}
		if len(point.Vector) != q.dimensions {
			return fmt.Errorf("qdrant point %s has dimension %d, want %d", point.ID, len(point.Vector), q.dimensions)
		}
		encoded[i] = map[string]any{
			"id": point.ID, "vector": point.Vector,
			"payload": map[string]any{
				"user_id": point.UserID, "learning_space_id": point.LearningSpaceID,
				"material_id": point.MaterialID, "chunk_id": point.ChunkID, "chunk_index": point.ChunkIndex,
			},
		}
	}
	status, response, err := q.do(ctx, http.MethodPut, "/collections/"+url.PathEscape(q.collection)+"/points?wait=true", map[string]any{"points": encoded})
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return qdrantStatusError("upsert points", status, response)
	}
	return nil
}

func (q *Qdrant) Search(ctx context.Context, vector []float32, userID, spaceID string, materialIDs []string, limit int) ([]SearchResult, error) {
	if len(vector) != q.dimensions {
		return nil, fmt.Errorf("query vector has dimension %d, want %d", len(vector), q.dimensions)
	}
	if userID == "" || spaceID == "" || limit <= 0 {
		return nil, fmt.Errorf("user ID, learning space ID, and positive limit are required")
	}
	must := []any{
		map[string]any{"key": "user_id", "match": map[string]any{"value": userID}},
		map[string]any{"key": "learning_space_id", "match": map[string]any{"value": spaceID}},
	}
	if len(materialIDs) > 0 {
		must = append(must, map[string]any{"key": "material_id", "match": map[string]any{"any": materialIDs}})
	}
	body := map[string]any{
		"query": vector, "limit": limit, "with_payload": true,
		"filter": map[string]any{"must": must},
	}
	status, response, err := q.do(ctx, http.MethodPost, "/collections/"+url.PathEscape(q.collection)+"/points/query", body)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, qdrantStatusError("query points", status, response)
	}
	var decoded struct {
		Result struct {
			Points []struct {
				ID      any            `json:"id"`
				Score   float32        `json:"score"`
				Payload map[string]any `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	if err = json.Unmarshal(response, &decoded); err != nil {
		return nil, fmt.Errorf("decode qdrant query response: %w", err)
	}
	results := make([]SearchResult, len(decoded.Result.Points))
	for i, point := range decoded.Result.Points {
		results[i] = SearchResult{PointID: fmt.Sprint(point.ID), Score: point.Score, Payload: point.Payload}
	}
	return results, nil
}

func (q *Qdrant) DeleteByMaterial(ctx context.Context, userID, spaceID, materialID string) error {
	if userID == "" || spaceID == "" || materialID == "" {
		return fmt.Errorf("user, learning space, and material IDs are required")
	}
	body := map[string]any{"filter": map[string]any{"must": []any{
		map[string]any{"key": "user_id", "match": map[string]any{"value": userID}},
		map[string]any{"key": "learning_space_id", "match": map[string]any{"value": spaceID}},
		map[string]any{"key": "material_id", "match": map[string]any{"value": materialID}},
	}}}
	status, response, err := q.do(ctx, http.MethodPost, "/collections/"+url.PathEscape(q.collection)+"/points/delete?wait=true", body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return qdrantStatusError("delete material points", status, response)
	}
	return nil
}

func (q *Qdrant) do(ctx context.Context, method, path string, value any) (int, []byte, error) {
	var body io.Reader
	if value != nil {
		encoded, err := json.Marshal(value)
		if err != nil {
			return 0, nil, fmt.Errorf("encode qdrant request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, body)
	if err != nil {
		return 0, nil, fmt.Errorf("create qdrant request: %w", err)
	}
	if value != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("call qdrant: %w", err)
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(io.LimitReader(resp.Body, maxQdrantResponseBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("read qdrant response: %w", err)
	}
	return resp.StatusCode, response, nil
}

func qdrantStatusError(operation string, status int, body []byte) error {
	return fmt.Errorf("qdrant %s returned status %d: %s", operation, status, strings.TrimSpace(string(body)))
}
