package ai

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// MockEmbedding is a deterministic local feature-hashing embedder for development
// and integration tests. It is useful for exercising Qdrant, not for evaluating
// retrieval quality or replacing a real embedding model.
type MockEmbedding struct{ dimensions int }

func NewMockEmbedding(dimensions int) (*MockEmbedding, error) {
	if dimensions <= 0 {
		return nil, fmt.Errorf("mock embedding dimensions must be positive")
	}
	return &MockEmbedding{dimensions: dimensions}, nil
}

func (m *MockEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		text = strings.TrimSpace(strings.ToLower(text))
		if text == "" {
			return nil, fmt.Errorf("embedding input must not contain empty text")
		}
		vector := make([]float32, m.dimensions)
		for _, token := range mockTokens(text) {
			h := fnv.New64a()
			_, _ = h.Write([]byte(token))
			value := h.Sum64()
			weight := float32(1)
			if value>>63 != 0 {
				weight = -1
			}
			vector[value%uint64(m.dimensions)] += weight
		}
		var norm float64
		for _, value := range vector {
			norm += float64(value * value)
		}
		if norm > 0 {
			scale := float32(1 / math.Sqrt(norm))
			for j := range vector {
				vector[j] *= scale
			}
		}
		result[i] = vector
	}
	return result, nil
}

func mockTokens(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSpace(r) })
	tokens := append([]string(nil), words...)
	runes := []rune(text)
	for i := 0; i+1 < len(runes); i++ {
		if !unicode.IsSpace(runes[i]) && !unicode.IsSpace(runes[i+1]) {
			tokens = append(tokens, string(runes[i:i+2]))
		}
	}
	return tokens
}
