package ai

import (
	"context"
	"strings"
	"testing"
)

func TestMockChatAnswerIncludesCitations(t *testing.T) {
	answer, err := (MockChatGenerator{}).Answer(context.Background(), "TCP 是什么", []ChatSource{{MaterialName: "book.pdf", Content: "TCP 是面向连接的可靠传输协议。"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(answer, "[1]") || !strings.Contains(answer, "TCP") {
		t.Fatalf("answer=%s", answer)
	}
}
