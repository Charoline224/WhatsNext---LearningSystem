package ai

import "context"

// Embedder isolates the application from a specific OpenAI-compatible provider.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
