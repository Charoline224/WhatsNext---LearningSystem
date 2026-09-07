package rag

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestQdrantUpsertCarriesIsolationPayload(t *testing.T) {
	client, err := NewQdrant("https://qdrant.test", "", "chunks", 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client.Transport = qdrantRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/collections/chunks/points" || r.URL.Query().Get("wait") != "true" {
			t.Fatalf("unexpected URL: %s", r.URL.String())
		}
		var request struct {
			Points []struct {
				Payload map[string]any `json:"payload"`
			} `json:"points"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		payload := request.Points[0].Payload
		if payload["user_id"] != "user" || payload["learning_space_id"] != "space" || payload["chunk_id"] != "chunk" {
			t.Fatalf("missing isolation payload: %v", payload)
		}
		return qdrantJSONResponse(http.StatusOK, `{"status":"ok"}`), nil
	})
	err = client.Upsert(context.Background(), []ChunkPoint{{ID: "point", Vector: []float32{1, 2}, UserID: "user", LearningSpaceID: "space", MaterialID: "material", ChunkID: "chunk"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestQdrantSearchAlwaysFiltersUserAndSpace(t *testing.T) {
	client, err := NewQdrant("https://qdrant.test", "", "chunks", 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client.Transport = qdrantRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request struct {
			Filter struct {
				Must []struct {
					Key string `json:"key"`
				} `json:"must"`
			} `json:"filter"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Filter.Must) != 2 || request.Filter.Must[0].Key != "user_id" || request.Filter.Must[1].Key != "learning_space_id" {
			t.Fatalf("unsafe filter: %+v", request.Filter.Must)
		}
		return qdrantJSONResponse(http.StatusOK, `{"result":{"points":[{"id":"point","score":0.9,"payload":{"chunk_id":"chunk"}}]}}`), nil
	})
	results, err := client.Search(context.Background(), []float32{1, 2}, "user", "space", nil, 5)
	if err != nil || len(results) != 1 || results[0].PointID != "point" {
		t.Fatalf("results=%v err=%v", results, err)
	}
}

func TestQdrantSearchAddsMaterialFilter(t *testing.T) {
	client, err := NewQdrant("https://qdrant.test", "", "chunks", 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	client.client.Transport = qdrantRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request struct {
			Filter struct {
				Must []struct {
					Key   string `json:"key"`
					Match struct {
						Any []string `json:"any"`
					} `json:"match"`
				} `json:"must"`
			} `json:"filter"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Filter.Must) != 3 || request.Filter.Must[2].Key != "material_id" || len(request.Filter.Must[2].Match.Any) != 2 {
			t.Fatalf("missing material filter: %+v", request.Filter.Must)
		}
		return qdrantJSONResponse(http.StatusOK, `{"result":{"points":[]}}`), nil
	})
	_, err = client.Search(context.Background(), []float32{1, 2}, "user", "space", []string{"m1", "m2"}, 5)
	if err != nil {
		t.Fatal(err)
	}
}

type qdrantRoundTripFunc func(*http.Request) (*http.Response, error)

func (f qdrantRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func qdrantJSONResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
