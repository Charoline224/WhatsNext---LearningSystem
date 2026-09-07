package dto

type KnowledgeNodeInput struct {
	Name             string   `json:"name"`
	NodeType         string   `json:"node_type"`
	Description      string   `json:"description"`
	ExamWeight       float64  `json:"exam_weight"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	PositionX        *float64 `json:"position_x"`
	PositionY        *float64 `json:"position_y"`
}

type KnowledgeNodePositionInput struct {
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
}
type KnowledgeEdgeInput struct {
	FromNodeID   string `json:"from_node_id"`
	ToNodeID     string `json:"to_node_id"`
	RelationType string `json:"relation_type"`
}
type KnowledgeArticleInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
