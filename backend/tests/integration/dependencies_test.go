//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/queue"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/storage"
)

func TestRealRedisMinIOAndQdrant(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	suffix := uuid.NewString()

	q := queue.NewRedisQueue(env("INTEGRATION_REDIS_ADDRESS", "localhost:6380"), "", "whatsnext:integration:"+suffix)
	defer q.Close()
	if err := q.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(ctx, "job-"+suffix); err != nil {
		t.Fatal(err)
	}
	if got, err := q.Dequeue(ctx, time.Second); err != nil || got != "job-"+suffix {
		t.Fatalf("redis dequeue=%q err=%v", got, err)
	}

	objects, err := storage.NewMinIO(env("INTEGRATION_MINIO_ENDPOINT", "localhost:9000"), env("INTEGRATION_MINIO_ACCESS_KEY", "whatsnext"), env("INTEGRATION_MINIO_SECRET_KEY", "whatsnext_minio_dev"), "whatsnext-integration", false)
	if err != nil {
		t.Fatal(err)
	}
	key := "tests/" + suffix + ".txt"
	defer objects.Delete(context.Background(), key)
	content := []byte("TCP 四次挥手集成测试")
	if err = objects.Put(ctx, key, "text/plain", bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatal(err)
	}
	if size, statErr := objects.Stat(ctx, key); statErr != nil || size != int64(len(content)) {
		t.Fatalf("minio size=%d err=%v", size, statErr)
	}
	if signed, signErr := objects.PresignedDownload(ctx, key, "test.txt", time.Minute); signErr != nil || signed == "" {
		t.Fatalf("minio signed URL err=%v", signErr)
	}

	embedder, _ := ai.NewMockEmbedding(64)
	vectors, err := embedder.Embed(ctx, []string{"为什么 TCP 需要四次挥手", "数据库事务隔离"})
	if err != nil {
		t.Fatal(err)
	}
	collection := "whatsnext_integration"
	vectorStore, err := rag.NewQdrant(env("INTEGRATION_QDRANT_URL", "http://localhost:6333"), "", collection, 64, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = vectorStore.EnsureCollection(ctx); err != nil {
		t.Fatal(err)
	}
	userID, spaceID := "user-"+suffix, "space-"+suffix
	points := []rag.ChunkPoint{
		{ID: uuid.NewString(), Vector: vectors[0], UserID: userID, LearningSpaceID: spaceID, MaterialID: "material-a", ChunkID: "chunk-a", ChunkIndex: 0},
		{ID: uuid.NewString(), Vector: vectors[1], UserID: "other-user", LearningSpaceID: spaceID, MaterialID: "material-b", ChunkID: "chunk-b", ChunkIndex: 0},
	}
	if err = vectorStore.Upsert(ctx, points); err != nil {
		t.Fatal(err)
	}
	results, err := vectorStore.Search(ctx, vectors[0], userID, spaceID, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || fmt.Sprint(results[0].Payload["chunk_id"]) != "chunk-a" {
		t.Fatalf("qdrant tenant filter leaked results: %+v", results)
	}
	if err = vectorStore.DeleteByMaterial(ctx, userID, spaceID, "material-a"); err != nil {
		t.Fatal(err)
	}
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
