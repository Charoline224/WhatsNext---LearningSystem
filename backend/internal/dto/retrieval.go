package dto

type RetrievalSearchInput struct {
	Query       string   `json:"query"`
	TopK        int      `json:"top_k"`
	MaterialIDs []string `json:"material_ids"`
}

type RetrievalResult struct {
	ChunkID      string  `json:"chunk_id"`
	MaterialID   string  `json:"material_id"`
	MaterialName string  `json:"material_name"`
	ChunkIndex   int     `json:"chunk_index"`
	Content      string  `json:"content"`
	SourceType   string  `json:"source_type"`
	SourceStart  *int    `json:"source_start"`
	SourceEnd    *int    `json:"source_end"`
	Score        float32 `json:"score"`
}

type RetrievalSearchResult struct {
	Items []RetrievalResult `json:"items"`
}
