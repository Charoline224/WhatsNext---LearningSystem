package repository

import (
	"strings"
	"testing"

	"whatsnext/backend/internal/ai"
)

func TestExamKnowledgeArticleBodyUsesAnalysis(t *testing.T) {
	body := examKnowledgeArticleBody(ai.GeneratedExamQuestion{
		PatternTitle:       "TCP 滑动窗口与流量控制",
		PatternDescription: "考察发送窗口、接收窗口与确认机制的联动关系。",
		TestedKnowledge:    "rwnd、cwnd、可用窗口和累计确认",
		SolvingStrategy:    "画出字节序号区间并标注各区域。",
	})

	for _, expected := range []string{
		"## 知识点概括",
		"rwnd、cwnd、可用窗口和累计确认",
		"TCP 滑动窗口与流量控制",
		"考察发送窗口、接收窗口与确认机制的联动关系。",
		"画出字节序号区间并标注各区域。",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("article body does not contain %q:\n%s", expected, body)
		}
	}
}

func TestExamKnowledgeArticleBodyHasUsefulFallbacks(t *testing.T) {
	body := examKnowledgeArticleBody(ai.GeneratedExamQuestion{})
	if strings.Contains(body, legacyExamKnowledgeArticleBody) {
		t.Fatalf("article body fell back to the legacy placeholder: %s", body)
	}
	for _, heading := range []string{"知识点概括", "在真题中如何考察", "理解与应用"} {
		if !strings.Contains(body, heading) {
			t.Fatalf("article body does not contain heading %q: %s", heading, body)
		}
	}
}
