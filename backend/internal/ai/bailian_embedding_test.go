package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBailianEmbeddingEmbedOrdersAndValidatesVectors(t *testing.T) {
	provider, err := NewBailianEmbedding("https://example.test/compatible-mode/v1/", "secret", "text-embedding-v4", 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/compatible-mode/v1/embeddings" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatal("missing authorization")
		}
		var request embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "text-embedding-v4" || request.Dimensions != 3 || len(request.Input) != 2 {
			t.Fatalf("unexpected request: %+v", request)
		}
		return jsonResponse(http.StatusOK, `{"data":[{"index":1,"embedding":[4,5,6]},{"index":0,"embedding":[1,2,3]}]}`), nil
	})
	vectors, err := provider.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if vectors[0][0] != 1 || vectors[1][0] != 4 {
		t.Fatalf("vectors are not ordered by input index: %v", vectors)
	}
}

func TestBailianEmbeddingRejectsWrongDimension(t *testing.T) {
	provider, err := NewBailianEmbedding("https://example.test", "secret", "model", 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"data":[{"index":0,"embedding":[1]}]}`), nil
	})
	if _, err = provider.Embed(context.Background(), []string{"text"}); err == nil {
		t.Fatal("expected dimension validation error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
