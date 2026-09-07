package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"whatsnext/backend/internal/model"
)

type BailianChat struct {
	endpoint, apiKey, generationModel, cheapModel string
	client                                        *http.Client
}

func NewBailianChat(baseURL, apiKey, generationModel, cheapModel string, timeout time.Duration) (*BailianChat, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(generationModel) == "" || strings.TrimSpace(cheapModel) == "" {
		return nil, fmt.Errorf("bailian base URL, API key, generation model, and cheap model are required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("generation timeout must be positive")
	}
	return &BailianChat{endpoint: baseURL + "/chat/completions", apiKey: apiKey, generationModel: generationModel, cheapModel: cheapModel, client: &http.Client{Timeout: timeout}}, nil
}

type chatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []chatCompletionMessage `json:"messages"`
	Temperature    float64                 `json:"temperature"`
	ResponseFormat *responseFormat         `json:"response_format,omitempty"`
}
type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type responseFormat struct {
	Type string `json:"type"`
}
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (b *BailianChat) complete(ctx context.Context, modelName, system, user string, structured bool) (string, error) {
	payload := chatCompletionRequest{Model: modelName, Messages: []chatCompletionMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}, Temperature: 0.2}
	if structured {
		payload.ResponseFormat = &responseFormat{Type: "json_object"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode chat completion request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call chat completion provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, maxProviderErrorBytes))
		return "", fmt.Errorf("chat completion provider returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	var decoded chatCompletionResponse
	if err = json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode chat completion response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("chat completion provider returned no content")
	}
	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}

func marshalPrompt(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode structured prompt: %w", err)
	}
	return string(data), nil
}
func decodeModelJSON(content string, target any) error {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), target); err != nil {
		return fmt.Errorf("decode model JSON: %w", err)
	}
	return nil
}

func (b *BailianChat) Answer(ctx context.Context, question string, sources []ChatSource) (string, error) {
	input, err := marshalPrompt(map[string]any{"question": question, "sources": sources})
	if err != nil {
		return "", err
	}
	return b.complete(ctx, b.generationModel, "你是学习助手。只根据提供的资料回答；资料不足时明确说明。引用资料时使用 [1]、[2] 编号，不编造来源。", input, false)
}

type knowledgeOutput struct {
	Nodes []struct {
		Name, Description, SourceChunkID string
		ExamWeight                       float64
		EstimatedMinutes                 int
	} `json:"nodes"`
	Edges []struct {
		From, To     int
		RelationType string
	} `json:"edges"`
	Articles []struct {
		Node                       int
		Title, Body, SourceChunkID string
	} `json:"articles"`
}

func (b *BailianChat) GenerateKnowledge(ctx context.Context, goal string, sources []AssetSource) (GeneratedAssets, error) {
	if len(sources) == 0 {
		return GeneratedAssets{}, fmt.Errorf("no indexed sources")
	}
	sources = sampleAssetSources(sources, 12)
	input, err := marshalPrompt(map[string]any{"goal": goal, "sources": sources})
	if err != nil {
		return GeneratedAssets{}, err
	}
	content, err := b.complete(ctx, b.generationModel, `根据资料生成“少而核心”的分层知识图和知识手册。先合并同义、上下位过细和可以在同一章节讲清的概念，只保留理解课程所必需的核心知识。知识点通常为 6-12 个，绝对不能超过 12 个；资料较少时可以更少。不要把学习目标、分数、具体题目、题型、解题步骤或考试年份作为知识节点。图必须是一个连通结构，每个节点都至少关联另一个节点；优先使用 contains 表达章节层级、prerequisite 表达学习顺序，只有确有直接概念联系时才使用 related，禁止为了凑边而建立弱关联。边数至少为节点数减一。

只输出 JSON 对象：{"nodes":[{"name":"","description":"","sourceChunkID":"原始chunk_id","examWeight":0.0,"estimatedMinutes":20}],"edges":[{"from":0,"to":1,"relationType":"prerequisite|related|contains"}],"articles":[{"node":0,"title":"","body":"","sourceChunkID":"原始chunk_id"}]}。索引必须有效，权重范围0到1，不得虚构 sourceChunkID。手册中的所有数学公式必须使用标准 LaTeX：行内公式使用 $...$，独立公式使用 $$...$$；指数必须使用 ^{...}，下标必须使用 _{...}，不要输出无定界符的裸公式。`, input, true)
	if err != nil {
		return GeneratedAssets{}, err
	}
	var out knowledgeOutput
	if err = decodeModelJSON(content, &out); err != nil {
		return GeneratedAssets{}, err
	}
	allowedSources := map[string]bool{}
	for _, source := range sources {
		allowedSources[source.ChunkID] = true
	}
	result := GeneratedAssets{}
	for _, node := range out.Nodes {
		if strings.TrimSpace(node.Name) == "" || !allowedSources[node.SourceChunkID] || node.ExamWeight < 0 || node.ExamWeight > 1 || node.EstimatedMinutes <= 0 {
			return GeneratedAssets{}, fmt.Errorf("model returned invalid knowledge node")
		}
		result.Nodes = append(result.Nodes, GeneratedNode{Name: node.Name, Description: node.Description, SourceChunkID: node.SourceChunkID, ExamWeight: node.ExamWeight, EstimatedMinutes: node.EstimatedMinutes})
	}
	if len(result.Nodes) == 0 || len(result.Nodes) > 12 {
		return GeneratedAssets{}, fmt.Errorf("model returned an invalid number of knowledge nodes")
	}
	for _, edge := range out.Edges {
		if edge.From < 0 || edge.From >= len(result.Nodes) || edge.To < 0 || edge.To >= len(result.Nodes) || (edge.RelationType != "prerequisite" && edge.RelationType != "related" && edge.RelationType != "contains") {
			return GeneratedAssets{}, fmt.Errorf("model returned invalid knowledge edge")
		}
		result.Edges = append(result.Edges, GeneratedEdge{From: edge.From, To: edge.To, RelationType: edge.RelationType})
	}
	if err = validateKnowledgeGraph(len(result.Nodes), result.Edges); err != nil {
		return GeneratedAssets{}, err
	}
	for _, article := range out.Articles {
		if article.Node < 0 || article.Node >= len(result.Nodes) || strings.TrimSpace(article.Body) == "" || !allowedSources[article.SourceChunkID] {
			return GeneratedAssets{}, fmt.Errorf("model returned invalid knowledge article")
		}
		result.Articles = append(result.Articles, GeneratedArticle{Node: article.Node, Title: article.Title, Body: article.Body, SourceChunkID: article.SourceChunkID})
	}
	return result, nil
}

func validateKnowledgeGraph(nodeCount int, edges []GeneratedEdge) error {
	if nodeCount <= 1 {
		return nil
	}
	if len(edges) < nodeCount-1 {
		return fmt.Errorf("model returned a weakly connected knowledge graph")
	}
	adjacency := make([][]int, nodeCount)
	seenEdges := make(map[[2]int]bool, len(edges))
	for _, edge := range edges {
		if edge.From == edge.To {
			return fmt.Errorf("model returned a self-referencing knowledge edge")
		}
		key := [2]int{edge.From, edge.To}
		if seenEdges[key] {
			return fmt.Errorf("model returned a duplicate knowledge edge")
		}
		seenEdges[key] = true
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		adjacency[edge.To] = append(adjacency[edge.To], edge.From)
	}
	visited := make([]bool, nodeCount)
	queue := []int{0}
	visited[0] = true
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	for _, connected := range visited {
		if !connected {
			return fmt.Errorf("model returned a disconnected knowledge graph")
		}
	}
	return nil
}

func sampleAssetSources(sources []AssetSource, limit int) []AssetSource {
	if len(sources) <= limit || limit <= 0 {
		return sources
	}
	if limit == 1 {
		return sources[:1]
	}
	result := make([]AssetSource, 0, limit)
	for i := range limit {
		index := i * (len(sources) - 1) / (limit - 1)
		result = append(result, sources[index])
	}
	return result
}

type planOutput struct {
	Title, Rationale string
	Stages           []struct {
		Node                       int
		Title, Description, Status string
		EstimatedDays              int
	} `json:"stages"`
	Tasks []struct {
		Node             int
		TaskType, Title  string
		EstimatedMinutes int
	} `json:"tasks"`
}

func (b *BailianChat) GeneratePlan(ctx context.Context, planContext PlanContext, sources []PlanSource, signals []PlanSignal) (GeneratedPlan, error) {
	if len(sources) == 0 {
		return GeneratedPlan{}, fmt.Errorf("knowledge map and handbook are required")
	}
	input, err := marshalPrompt(map[string]any{"context": planContext, "nodes": sources, "signals": signals})
	if err != nil {
		return GeneratedPlan{}, err
	}
	content, err := b.complete(ctx, b.generationModel, `你是个性化学习规划 Agent。根据用户上传资料提炼的 nodes、考试时间、每日时间、掌握度、置信度、答题证据与 signals，同时生成整体路线和今日任务。路线必须使用资料中的具体知识点，按弱项、考试权重、前置基础和剩余时间排序，禁止输出“资料建模/系统学习/真题校准/考前收束”这类通用流程。只输出 JSON 对象：{"title":"针对当前目标的具体计划名","rationale":"说明用了哪些资料与学情信号","stages":[{"node":0,"title":"具体阶段名","description":"该用户为何先学什么、如何验收","status":"pending|active|complete","estimatedDays":2}],"tasks":[{"node":0,"taskType":"learn|review|practice|test","title":"具体任务","estimatedMinutes":20}]}。node 是输入 nodes 的数组索引；stages 为 1 至 6 个、每阶段必须绑定一个具体知识点；title 不超过 40 字，rationale 不超过 180 字，阶段 title 不超过 40 字且 description 不超过 220 字；今日任务总时长不得超过 context.dailyMinutes。`, input, true)
	if err != nil {
		return GeneratedPlan{}, err
	}
	var out planOutput
	if err = decodeModelJSON(content, &out); err != nil {
		return GeneratedPlan{}, err
	}
	if strings.TrimSpace(out.Title) == "" || utf8.RuneCountInString(out.Title) > 255 || strings.TrimSpace(out.Rationale) == "" || utf8.RuneCountInString(out.Rationale) > 500 || len(out.Stages) == 0 || len(out.Stages) > 6 {
		return GeneratedPlan{}, fmt.Errorf("model returned an incomplete personalized roadmap")
	}
	stages := make([]GeneratedRoadmapStage, 0, len(out.Stages))
	for _, stage := range out.Stages {
		if stage.Node < 0 || stage.Node >= len(sources) || strings.TrimSpace(stage.Title) == "" || utf8.RuneCountInString(stage.Title) > 255 || strings.TrimSpace(stage.Description) == "" || utf8.RuneCountInString(stage.Description) > 1000 || stage.EstimatedDays <= 0 || (stage.Status != "pending" && stage.Status != "active" && stage.Status != "complete") {
			return GeneratedPlan{}, fmt.Errorf("model returned an invalid personalized roadmap stage")
		}
		stages = append(stages, GeneratedRoadmapStage{Node: stage.Node, Title: stage.Title, Description: stage.Description, Status: stage.Status, EstimatedDays: stage.EstimatedDays})
	}
	result, total := make([]GeneratedTask, 0, len(out.Tasks)), 0
	for _, task := range out.Tasks {
		if task.Node < 0 || task.Node >= len(sources) || task.EstimatedMinutes <= 0 || strings.TrimSpace(task.Title) == "" || utf8.RuneCountInString(task.Title) > 255 || (task.TaskType != "learn" && task.TaskType != "review" && task.TaskType != "practice" && task.TaskType != "test") {
			return GeneratedPlan{}, fmt.Errorf("model returned invalid learning task")
		}
		total += task.EstimatedMinutes
		result = append(result, GeneratedTask{Node: task.Node, TaskType: task.TaskType, Title: task.Title, EstimatedMinutes: task.EstimatedMinutes})
	}
	if len(result) == 0 || total > planContext.DailyMinutes {
		return GeneratedPlan{}, fmt.Errorf("model returned an empty or over-budget learning plan")
	}
	return GeneratedPlan{Title: out.Title, Rationale: out.Rationale, Stages: stages, Tasks: result}, nil
}

type examOutput struct {
	Patterns []struct {
		PatternKey, PatternTitle, PatternDescription, TestedKnowledge, CommonMistakes, SolvingStrategy string
		KnowledgeNames                                                                                 []string
	} `json:"patterns"`
	Questions []struct {
		Chunk              int
		Stem, QuestionType string
		Pattern            int
	} `json:"questions"`
}

func (b *BailianChat) AnalyzeExam(ctx context.Context, chunks []model.MaterialChunk) ([]GeneratedExamQuestion, error) {
	if len(chunks) == 0 {
		return []GeneratedExamQuestion{}, nil
	}
	if len(chunks) > 20 {
		chunks = chunks[:20]
	}
	type examSource struct {
		Index               int `json:"index"`
		Content, SourceType string
		SourceStart         *int
	}
	sources := make([]examSource, len(chunks))
	for i, chunk := range chunks {
		sources[i] = examSource{Index: i, Content: chunk.Content, SourceType: chunk.SourceType, SourceStart: chunk.SourceStart}
	}
	input, err := marshalPrompt(map[string]any{"chunks": sources})
	if err != nil {
		return nil, err
	}
	content, err := b.complete(ctx, b.cheapModel, `你是试卷题型分析专家。先识别全部题目，再从整份试卷的全局视角归纳有限的题型簇，最后将每道题归入一个题型。题型必须按共同的解题方法、知识结构和易错模式归并，不能复述某一道具体题干，也不能仅把单个考点当作题型。相同解法的题必须复用同一 pattern 索引；只要存在多道题，就必须至少有一个题型包含多道题。题型数量通常为题目数的 20%-50%，最多 10 个。patternKey 使用稳定、粗粒度的英文 snake_case 标识，使不同年份同类试卷也能复用。

只输出 JSON 对象：{"patterns":[{"patternKey":"first_order_linear_ode","patternTitle":"一阶线性微分方程求解","patternDescription":"识别标准形式并用积分因子求通解","testedKnowledge":"","commonMistakes":"","solvingStrategy":"","knowledgeNames":[""]}],"questions":[{"chunk":0,"stem":"","questionType":"choice|calculation|proof|short_answer","pattern":0}]}。questions.pattern 必须引用 patterns 的数组索引，chunk 必须引用输入索引。`, input, true)
	if err != nil {
		return nil, err
	}
	var out examOutput
	if err = decodeModelJSON(content, &out); err != nil {
		return nil, err
	}
	if len(out.Patterns) == 0 || len(out.Questions) == 0 || len(out.Patterns) > len(out.Questions) {
		return nil, fmt.Errorf("model returned invalid exam patterns")
	}
	result := make([]GeneratedExamQuestion, 0, len(out.Questions))
	patternUsage := make([]int, len(out.Patterns))
	for _, question := range out.Questions {
		if question.Chunk < 0 || question.Chunk >= len(chunks) || strings.TrimSpace(question.Stem) == "" || question.Pattern < 0 || question.Pattern >= len(out.Patterns) {
			return nil, fmt.Errorf("model returned invalid exam question")
		}
		pattern := out.Patterns[question.Pattern]
		if strings.TrimSpace(pattern.PatternKey) == "" || strings.TrimSpace(pattern.PatternTitle) == "" {
			return nil, fmt.Errorf("model returned invalid exam pattern")
		}
		patternUsage[question.Pattern]++
		source := chunks[question.Chunk]
		result = append(result, GeneratedExamQuestion{Stem: question.Stem, QuestionType: question.QuestionType, PatternKey: pattern.PatternKey, PatternTitle: pattern.PatternTitle, PatternDescription: pattern.PatternDescription, TestedKnowledge: pattern.TestedKnowledge, CommonMistakes: pattern.CommonMistakes, SolvingStrategy: pattern.SolvingStrategy, KnowledgeNames: pattern.KnowledgeNames, SourceType: source.SourceType, SourceStart: source.SourceStart})
	}
	if len(result) > 1 {
		shared := false
		for _, usage := range patternUsage {
			if usage > 1 {
				shared = true
				break
			}
		}
		if !shared {
			return nil, fmt.Errorf("model did not summarize shared exam patterns")
		}
	}
	return result, nil
}
