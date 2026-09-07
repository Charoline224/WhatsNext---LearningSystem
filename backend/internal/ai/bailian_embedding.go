package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxProviderErrorBytes = 4096

type BailianEmbedding struct {
	endpoint   string
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

func NewBailianEmbedding(baseURL, apiKey, model string, dimensions int, timeout time.Duration) (*BailianEmbedding, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, fmt.Errorf("bailian base URL, API key, and embedding model are required")
	}
	if dimensions <= 0 || timeout <= 0 {
		return nil, fmt.Errorf("embedding dimensions and timeout must be positive")
	}
	return &BailianEmbedding{
		endpoint:   baseURL + "/embeddings",
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		client:     &http.Client{Timeout: timeout},
	}, nil
}

type embeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (b *BailianEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	for _, value := range texts {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("embedding input must not contain empty text")
		}
	}
	body, err := json.Marshal(embeddingRequest{Model: b.model, Input: texts, Dimensions: b.dimensions})
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, maxProviderErrorBytes))
		return nil, fmt.Errorf("embedding provider returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	var decoded embeddingResponse
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decoded.Data) != len(texts) {
		return nil, fmt.Errorf("embedding provider returned %d vectors for %d inputs", len(decoded.Data), len(texts))
	}
	vectors := make([][]float32, len(texts))
	seen := make([]bool, len(texts))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(texts) || seen[item.Index] {
			return nil, fmt.Errorf("embedding provider returned invalid index %d", item.Index)
		}
		if len(item.Embedding) != b.dimensions {
			return nil, fmt.Errorf("embedding at index %d has dimension %d, want %d", item.Index, len(item.Embedding), b.dimensions)
		}
		vectors[item.Index] = item.Embedding
		seen[item.Index] = true
	}
	return vectors, nil
}
