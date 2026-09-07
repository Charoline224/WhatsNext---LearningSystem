package dto

type ChatInput struct {
	Message     string   `json:"message"`
	MaterialIDs []string `json:"material_ids"`
}
type ChatResult struct {
	Answer         string            `json:"answer"`
	Sources        []RetrievalResult `json:"sources"`
	DecisionSignal DecisionSignal    `json:"decision_signal"`
}

type DecisionSignal struct {
	ID            string  `json:"id"`
	RelatedNodeID *string `json:"related_node_id"`
	Status        string  `json:"status"`
}
