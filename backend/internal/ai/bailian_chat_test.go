package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"whatsnext/backend/internal/model"
)

func TestBailianChatAnswerUsesCompatibleEndpoint(t *testing.T) {
	provider, err := NewBailianChat("https://example.test/compatible-mode/v1/", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization header missing")
		}
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "qwen-plus" || len(request.Messages) != 2 {
			t.Fatalf("unexpected request: %+v", request)
		}
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"根据资料回答 [1]"}}]}`), nil
	})
	answer, err := provider.Answer(context.Background(), "问题", []ChatSource{{MaterialName: "book.pdf", Content: "资料"}})
	if err != nil {
		t.Fatal(err)
	}
	if answer != "根据资料回答 [1]" {
		t.Fatalf("answer = %q", answer)
	}
}

func TestBailianChatAnalyzeExamReusesSummarizedPattern(t *testing.T) {
	provider, err := NewBailianChat("https://example.test", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "qwen-flash" {
			t.Fatalf("exam analysis model = %q", request.Model)
		}
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"patterns\":[{\"patternKey\":\"definite_integral_calculation\",\"patternTitle\":\"定积分计算\",\"patternDescription\":\"选择合适方法计算定积分\",\"testedKnowledge\":\"换元与分部积分\",\"commonMistakes\":\"漏换积分限\",\"solvingStrategy\":\"先识别结构再选方法\",\"knowledgeNames\":[\"定积分\"]}],\"questions\":[{\"chunk\":0,\"stem\":\"计算积分一\",\"questionType\":\"calculation\",\"pattern\":0},{\"chunk\":1,\"stem\":\"计算积分二\",\"questionType\":\"calculation\",\"pattern\":0}]}"}}]}`), nil
	})
	page := 3
	questions, err := provider.AnalyzeExam(context.Background(), []model.MaterialChunk{
		{Content: "题目一", SourceType: "page", SourceStart: &page},
		{Content: "题目二", SourceType: "page", SourceStart: &page},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 || questions[0].PatternKey != questions[1].PatternKey {
		t.Fatalf("questions were not grouped: %+v", questions)
	}
	if questions[0].PatternTitle != "定积分计算" || questions[0].SourceStart == nil || *questions[0].SourceStart != page {
		t.Fatalf("pattern or source metadata was not mapped: %+v", questions[0])
	}
}

func TestBailianChatAnalyzeExamRejectsOnePatternPerQuestion(t *testing.T) {
	provider, err := NewBailianChat("https://example.test", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"patterns\":[{\"patternKey\":\"p1\",\"patternTitle\":\"题型一\"},{\"patternKey\":\"p2\",\"patternTitle\":\"题型二\"}],\"questions\":[{\"chunk\":0,\"stem\":\"题一\",\"questionType\":\"short_answer\",\"pattern\":0},{\"chunk\":1,\"stem\":\"题二\",\"questionType\":\"short_answer\",\"pattern\":1}]}"}}]}`), nil
	})
	_, err = provider.AnalyzeExam(context.Background(), []model.MaterialChunk{{Content: "题一"}, {Content: "题二"}})
	if err == nil {
		t.Fatal("expected one-pattern-per-question output to be rejected")
	}
}

func TestBailianChatAnalyzeExamAllowsMoreThanTenSharedPatterns(t *testing.T) {
	provider, err := NewBailianChat("https://example.test", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	patterns := make([]map[string]any, 11)
	questions := make([]map[string]any, 12)
	for i := range patterns {
		patterns[i] = map[string]any{"patternKey": fmt.Sprintf("pattern_%d", i), "patternTitle": fmt.Sprintf("题型%d", i)}
	}
	for i := range questions {
		pattern := i
		if pattern == 11 {
			pattern = 0
		}
		questions[i] = map[string]any{"chunk": 0, "stem": fmt.Sprintf("题目%d", i), "questionType": "calculation", "pattern": pattern}
	}
	output, err := json.Marshal(map[string]any{"patterns": patterns, "questions": questions})
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": string(output)}}}})
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, string(response)), nil
	})
	result, err := provider.AnalyzeExam(context.Background(), []model.MaterialChunk{{Content: "整份试卷"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 12 || result[0].PatternKey != result[11].PatternKey {
		t.Fatalf("unexpected clustered result: %+v", result)
	}
}

func TestBailianChatGeneratePlanValidatesBudget(t *testing.T) {
	provider, err := NewBailianChat("https://example.test", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"content":"{\"title\":\"TCP 冲刺\",\"rationale\":\"基于课件与掌握度\",\"stages\":[{\"node\":0,\"title\":\"攻克 TCP\",\"description\":\"先补齐可靠传输\",\"status\":\"active\",\"estimatedDays\":1}],\"tasks\":[{\"node\":0,\"taskType\":\"learn\",\"title\":\"学习 TCP\",\"estimatedMinutes\":40}]}"}}]}`), nil
	})
	_, err = provider.GeneratePlan(context.Background(), PlanContext{Goal: "通过考试", DailyMinutes: 30}, []PlanSource{{NodeID: "node", Name: "TCP"}}, nil)
	if err == nil {
		t.Fatal("expected over-budget plan error")
	}
}

func TestBailianChatRejectsProviderError(t *testing.T) {
	provider, err := NewBailianChat("https://example.test", "secret", "qwen-plus", "qwen-flash", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	provider.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, `{"error":{"message":"invalid request"}}`), nil
	})
	if _, err = provider.Answer(context.Background(), "问题", nil); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestSampleAssetSourcesCoversWholeMaterial(t *testing.T) {
	sources := make([]AssetSource, 100)
	for i := range sources {
		sources[i].ChunkID = string(rune(i))
	}
	selected := sampleAssetSources(sources, 12)
	if len(selected) != 12 {
		t.Fatalf("selected %d sources", len(selected))
	}
	if selected[0].ChunkID != sources[0].ChunkID || selected[len(selected)-1].ChunkID != sources[len(sources)-1].ChunkID {
		t.Fatalf("selection does not cover first and last chunks")
	}
}
