package repository

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
)

type ChatRepository struct{ db *sqlx.DB }

func NewChatRepository(db *sqlx.DB) *ChatRepository { return &ChatRepository{db: db} }

func (r *ChatRepository) SaveSignal(ctx context.Context, userID, spaceID, question, answer string, sources []dto.RetrievalResult) (model.ChatLearningSignal, error) {
	signal := model.ChatLearningSignal{ID: newRepositoryID(), Question: question, Answer: answer}
	sourcesJSON, err := json.Marshal(sources)
	if err != nil {
		return signal, err
	}
	if len(sources) > 0 {
		var nodeID string
		err = r.db.GetContext(ctx, &nodeID, `SELECT id FROM learning_nodes WHERE user_id=? AND learning_space_id=? AND source_chunk_id=? ORDER BY sort_order LIMIT 1`, userID, spaceID, sources[0].ChunkID)
		if err == nil {
			signal.RelatedNodeID = &nodeID
		}
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO chat_learning_signals(id,user_id,learning_space_id,question,answer,sources_json,related_node_id) VALUES(?,?,?,?,?,?,?)`, signal.ID, userID, spaceID, question, answer, sourcesJSON, signal.RelatedNodeID)
	return signal, err
}

func (r *ChatRepository) List(ctx context.Context, userID, spaceID string) ([]model.ChatLearningSignal, error) {
	items := []model.ChatLearningSignal{}
	err := r.db.SelectContext(ctx, &items, `SELECT id,question,answer,related_node_id,incorporated_plan_id,created_at FROM chat_learning_signals WHERE user_id=? AND learning_space_id=? ORDER BY created_at`, userID, spaceID)
	return items, err
}
