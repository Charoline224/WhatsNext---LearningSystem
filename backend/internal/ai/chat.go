package ai

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type ChatSource struct {
	MaterialName, Content string
	SourceStart           *int
}
type ChatGenerator interface {
	Answer(context.Context, string, []ChatSource) (string, error)
}
type MockChatGenerator struct{}
type UnavailableChatGenerator struct{}

func (UnavailableChatGenerator) Answer(context.Context, string, []ChatSource) (string, error) {
	return "", fmt.Errorf("chat generation provider is not configured")
}
func (MockChatGenerator) Answer(ctx context.Context, question string, sources []ChatSource) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(sources) == 0 {
		return "当前已索引资料中没有找到足够相关的内容。你可以换一种问法，或先上传更完整的学习资料。", nil
	}
	if len(sources) > 3 {
		sources = sources[:3]
	}
	parts := []string{fmt.Sprintf("针对“%s”，根据当前学习资料：", strings.TrimSpace(question))}
	for i, source := range sources {
		content := strings.Join(strings.Fields(source.Content), " ")
		if utf8.RuneCountInString(content) > 220 {
			content = string([]rune(content)[:220]) + "…"
		}
		parts = append(parts, fmt.Sprintf("[%d] %s", i+1, content))
	}
	parts = append(parts, "建议结合下方引用回到原始资料核对关键定义和细节。")
	return strings.Join(parts, "\n\n"), nil
}
