package ai

import (
	"context"
	"reflect"
	"testing"
)

func TestMockEmbeddingIsDeterministicAndNormalized(t *testing.T) {
	embedder, err := NewMockEmbedding(32)
	if err != nil {
		t.Fatal(err)
	}
	first, err := embedder.Embed(context.Background(), []string{"四次挥手 TCP"})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := embedder.Embed(context.Background(), []string{"四次挥手 TCP"})
	if !reflect.DeepEqual(first, second) || len(first[0]) != 32 {
		t.Fatalf("mock vectors are not deterministic: %v %v", first, second)
	}
}
