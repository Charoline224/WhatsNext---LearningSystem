package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"whatsnext/backend/internal/model"
)

type GeneratedExamQuestion struct {
	Stem, QuestionType, PatternKey, PatternTitle, PatternDescription, TestedKnowledge, CommonMistakes, SolvingStrategy, SourceType string
	KnowledgeNames                                                                                                                 []string
	SourceStart                                                                                                                    *int
}
type ExamAnalyzer interface {
	AnalyzeExam(context.Context, []model.MaterialChunk) ([]GeneratedExamQuestion, error)
}
type MockExamAnalyzer struct{}
type UnavailableExamAnalyzer struct{}

func (UnavailableExamAnalyzer) AnalyzeExam(context.Context, []model.MaterialChunk) ([]GeneratedExamQuestion, error) {
	return nil, fmt.Errorf("exam analysis provider is not configured")
}

var questionStart = regexp.MustCompile(`(?m)^\s*(?:\d{1,3}[.、)]|[(（]\d{1,3}[)）])\s*`)

func (MockExamAnalyzer) AnalyzeExam(ctx context.Context, chunks []model.MaterialChunk) ([]GeneratedExamQuestion, error) {
	result := []GeneratedExamQuestion{}
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		locations := questionStart.FindAllStringIndex(chunk.Content, -1)
		if len(locations) == 0 {
			continue
		}
		for i, location := range locations {
			end := len(chunk.Content)
			if i+1 < len(locations) {
				end = locations[i+1][0]
			}
			stem := strings.TrimSpace(chunk.Content[location[1]:end])
			if stem == "" {
				continue
			}
			qType := classifyQuestionType(stem)
			key, title, desc, knowledge, mistakes, strategy, names := analyzePattern(stem)
			result = append(result, GeneratedExamQuestion{Stem: stem, QuestionType: qType, PatternKey: key, PatternTitle: title, PatternDescription: desc, TestedKnowledge: knowledge, CommonMistakes: mistakes, SolvingStrategy: strategy, KnowledgeNames: names, SourceType: chunk.SourceType, SourceStart: chunk.SourceStart})
		}
	}
	if len(result) == 0 && len(chunks) > 0 {
		qType := classifyQuestionType(chunks[0].Content)
		key, title, desc, knowledge, mistakes, strategy, names := analyzePattern(chunks[0].Content)
		result = append(result, GeneratedExamQuestion{Stem: strings.TrimSpace(chunks[0].Content), QuestionType: qType, PatternKey: key, PatternTitle: title, PatternDescription: desc, TestedKnowledge: knowledge, CommonMistakes: mistakes, SolvingStrategy: strategy, KnowledgeNames: names, SourceType: chunks[0].SourceType, SourceStart: chunks[0].SourceStart})
	}
	return result, nil
}
func classifyQuestionType(stem string) string {
	lower := strings.ToLower(stem)
	switch {
	case strings.Contains(stem, "计算") || strings.Contains(lower, "calculate"):
		return "calculation"
	case strings.Contains(stem, "选择") || strings.Contains(stem, "A.") || strings.Contains(stem, "A、"):
		return "choice"
	case strings.Contains(stem, "证明") || strings.Contains(lower, "prove"):
		return "proof"
	default:
		return "short_answer"
	}
}
func analyzePattern(stem string) (string, string, string, string, string, string, []string) {
	switch {
	case strings.Contains(stem, "TIME_WAIT") || strings.Contains(stem, "四次挥手"):
		return "tcp_connection_close", "TCP 连接释放与 TIME_WAIT", "围绕四次挥手的状态迁移、报文方向和 TIME_WAIT 设计理由展开。", "TCP 连接释放、半关闭、2MSL 与旧报文消失", "把建立连接的三次握手与释放连接混淆；忽略主动关闭方进入 TIME_WAIT", "先标出主动关闭方，再按 FIN、ACK、FIN、ACK 的方向画状态图，最后解释 2MSL", []string{"TCP 连接释放", "TIME_WAIT"}
	case strings.Contains(stem, "滑动窗口"):
		return "tcp_sliding_window", "TCP 滑动窗口与流量控制", "考察发送窗口、接收窗口与确认机制的联动关系。", "rwnd、cwnd、可用窗口和累计确认", "将流量控制与拥塞控制混淆；没有扣除已发送未确认的字节", "画出字节序号区间，分别标注已确认、已发送和尚可发送区域", []string{"TCP 滑动窗口", "TCP 流量控制"}
	case strings.Contains(stem, "RTT") || strings.Contains(stem, "拥塞窗口"):
		return "tcp_window_calculation", "TCP 窗口与吞吐量计算", "结合 RTT、接收窗口和拥塞窗口计算可发送数据量或理论吞吐量。", "RTT、rwnd、cwnd、带宽时延积", "直接使用带宽而忽略窗口上限；单位换算错误", "先确定 min(rwnd,cwnd) 再结合 RTT；统一 bit、byte 和时间单位", []string{"TCP 拥塞控制", "带宽时延积"}
	case strings.Contains(stem, "可靠传输") || strings.Contains(stem, "序号") || strings.Contains(stem, "确认"):
		return "tcp_reliability", "TCP 可靠传输机制", "识别序号、确认、超时重传与校验和如何协同保证可靠性。", "seq/ack 语义、超时重传、累计确认", "认为只有校验和就能恢复丢包；混淆确认号与已收到序号", "先区分‘检测错误’与‘恢复错误’，再逐项对应 TCP 机制", []string{"TCP 可靠传输", "TCP 确认与重传"}
	default:
		return "concept_application", "核心概念理解与应用", "根据题干语境识别核心概念，并用完整逻辑解释。", "定义、适用条件与机制联系", "只背结论而忽略条件；答案缺少因果链", "先写定义，再列条件与机制，最后回到题干给出结论", []string{"核心概念"}
	}
}
