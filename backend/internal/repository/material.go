package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/model"
)

const materialColumns = `id,user_id,learning_space_id,material_kind,original_name,object_key,mime_type,size_bytes,content_sha256,status,failure_reason,created_at,updated_at`
const jobColumns = `id,user_id,learning_space_id,material_id,job_type,status,progress,attempts,max_attempts,available_at,started_at,completed_at,error_code,error_message,created_at,updated_at`
const chunkColumns = `id,user_id,learning_space_id,material_id,chunk_index,content,source_type,source_start,source_end,char_count,token_estimate,content_hash,created_at`
const legacyExamKnowledgeArticleBody = "该知识点由真题题型分析识别，建议结合题型手册和原始资料完成理解与练习。"

type MaterialRepository struct{ db *sqlx.DB }

func NewMaterialRepository(db *sqlx.DB) *MaterialRepository { return &MaterialRepository{db: db} }
func (r *MaterialRepository) CreateWithJob(ctx context.Context, m model.LearningMaterial, j model.GenerationJob) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin material transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_materials(id,user_id,learning_space_id,material_kind,original_name,object_key,mime_type,size_bytes,content_sha256,status) VALUES(?,?,?,?,?,?,?,?,?,?)`, m.ID, m.UserID, m.LearningSpaceID, m.MaterialKind, m.OriginalName, m.ObjectKey, m.MIMEType, m.SizeBytes, m.ContentSHA256, m.Status)
	if err != nil {
		return fmt.Errorf("insert material: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO generation_jobs(id,user_id,learning_space_id,material_id,job_type,status,progress,attempts,max_attempts,available_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, j.ID, j.UserID, j.LearningSpaceID, j.MaterialID, j.JobType, j.Status, j.Progress, j.Attempts, j.MaxAttempts, j.AvailableAt)
	if err != nil {
		return fmt.Errorf("insert generation job: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit material transaction: %w", err)
	}
	return nil
}
func (r *MaterialRepository) GetMaterial(ctx context.Context, userID, id string) (model.LearningMaterial, error) {
	var m model.LearningMaterial
	err := r.db.GetContext(ctx, &m, `SELECT `+materialColumns+` FROM learning_materials WHERE id=? AND user_id=?`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.LearningMaterial{}, ErrNotFound
	}
	if err != nil {
		return model.LearningMaterial{}, fmt.Errorf("get material: %w", err)
	}
	return m, nil
}
func (r *MaterialRepository) ListMaterials(ctx context.Context, userID, spaceID string) ([]model.LearningMaterial, error) {
	items := make([]model.LearningMaterial, 0)
	err := r.db.SelectContext(ctx, &items, `SELECT `+materialColumns+` FROM learning_materials WHERE user_id=? AND learning_space_id=? ORDER BY created_at DESC`, userID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	return items, nil
}
func (r *MaterialRepository) GetJob(ctx context.Context, userID, spaceID, id string) (model.GenerationJob, error) {
	var j model.GenerationJob
	err := r.db.GetContext(ctx, &j, `SELECT `+jobColumns+` FROM generation_jobs WHERE id=? AND user_id=? AND learning_space_id=?`, id, userID, spaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.GenerationJob{}, ErrNotFound
	}
	if err != nil {
		return model.GenerationJob{}, fmt.Errorf("get job: %w", err)
	}
	return j, nil
}
func (r *MaterialRepository) GetJobByID(ctx context.Context, id string) (model.GenerationJob, error) {
	var j model.GenerationJob
	err := r.db.GetContext(ctx, &j, `SELECT `+jobColumns+` FROM generation_jobs WHERE id=?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.GenerationJob{}, ErrNotFound
	}
	if err != nil {
		return model.GenerationJob{}, fmt.Errorf("get worker job: %w", err)
	}
	return j, nil
}
func (r *MaterialRepository) GetLatestJobForMaterial(ctx context.Context, userID, spaceID, materialID string) (model.GenerationJob, error) {
	var j model.GenerationJob
	err := r.db.GetContext(ctx, &j, `SELECT `+jobColumns+` FROM generation_jobs WHERE user_id=? AND learning_space_id=? AND material_id=? AND job_type='process_material' ORDER BY created_at DESC LIMIT 1`, userID, spaceID, materialID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.GenerationJob{}, ErrNotFound
	}
	if err != nil {
		return model.GenerationJob{}, fmt.Errorf("get material job: %w", err)
	}
	return j, nil
}

func (r *MaterialRepository) GetLatestEmbeddingJobForMaterial(ctx context.Context, userID, spaceID, materialID string) (model.GenerationJob, error) {
	var j model.GenerationJob
	err := r.db.GetContext(ctx, &j, `SELECT `+jobColumns+` FROM generation_jobs WHERE user_id=? AND learning_space_id=? AND material_id=? AND job_type='embed_material' ORDER BY created_at DESC LIMIT 1`, userID, spaceID, materialID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.GenerationJob{}, ErrNotFound
	}
	if err != nil {
		return model.GenerationJob{}, fmt.Errorf("get material embedding job: %w", err)
	}
	return j, nil
}
func (r *MaterialRepository) GetMaterialByID(ctx context.Context, id string) (model.LearningMaterial, error) {
	var m model.LearningMaterial
	err := r.db.GetContext(ctx, &m, `SELECT `+materialColumns+` FROM learning_materials WHERE id=?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.LearningMaterial{}, ErrNotFound
	}
	if err != nil {
		return model.LearningMaterial{}, fmt.Errorf("get worker material: %w", err)
	}
	return m, nil
}

func (r *MaterialRepository) ClaimJob(ctx context.Context, id string) (model.GenerationJob, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.GenerationJob{}, false, fmt.Errorf("begin claim job: %w", err)
	}
	defer tx.Rollback()
	var j model.GenerationJob
	err = tx.GetContext(ctx, &j, `SELECT `+jobColumns+` FROM generation_jobs WHERE id=? FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.GenerationJob{}, false, nil
	}
	if err != nil {
		return model.GenerationJob{}, false, fmt.Errorf("select job for claim: %w", err)
	}
	now := time.Now().UTC()
	if j.Status != "queued" || j.AvailableAt.After(now) || j.Attempts >= j.MaxAttempts {
		return model.GenerationJob{}, false, nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE generation_jobs SET status='processing',progress=10,attempts=attempts+1,started_at=?,error_code=NULL,error_message=NULL WHERE id=?`, now, id)
	if err != nil {
		return model.GenerationJob{}, false, fmt.Errorf("claim job: %w", err)
	}
	if j.JobType == "process_material" {
		_, err = tx.ExecContext(ctx, `UPDATE learning_materials SET status='processing',failure_reason=NULL WHERE id=?`, j.MaterialID)
		if err != nil {
			return model.GenerationJob{}, false, fmt.Errorf("mark material processing: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return model.GenerationJob{}, false, fmt.Errorf("commit claim job: %w", err)
	}
	j.Status = "processing"
	j.Progress = 10
	j.Attempts++
	j.StartedAt = &now
	return j, true, nil
}
func (r *MaterialRepository) NextQueuedJobID(ctx context.Context) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id, `SELECT id FROM generation_jobs WHERE status='queued' AND available_at<=? AND attempts<max_attempts ORDER BY available_at,created_at LIMIT 1`, time.Now().UTC())
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("find queued job: %w", err)
	}
	return id, nil
}

// RequeueInterruptedJobs recovers work left in processing when the sole worker
// was stopped or restarted. An interrupted attempt does not consume retry budget.
func (r *MaterialRepository) RequeueInterruptedJobs(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE generation_jobs SET status='queued',progress=0,attempts=GREATEST(attempts-1,0),available_at=?,started_at=NULL,error_code='WORKER_INTERRUPTED',error_message='Worker restarted before the job completed' WHERE status='processing'`, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("requeue interrupted material jobs: %w", err)
	}
	return result.RowsAffected()
}
func (r *MaterialRepository) CompleteJob(ctx context.Context, j model.GenerationJob, chunks []model.MaterialChunk, embeddingJob model.GenerationJob, embeddings []model.ChunkEmbedding, questions []ai.GeneratedExamQuestion) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin complete job: %w", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `DELETE FROM material_chunks WHERE material_id=?`, j.MaterialID); err != nil {
		return fmt.Errorf("replace material chunks: %w", err)
	}
	for _, chunk := range chunks {
		_, err = tx.ExecContext(ctx, `INSERT INTO material_chunks(id,user_id,learning_space_id,material_id,chunk_index,content,source_type,source_start,source_end,char_count,token_estimate,content_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, chunk.ID, chunk.UserID, chunk.LearningSpaceID, chunk.MaterialID, chunk.ChunkIndex, chunk.Content, chunk.SourceType, chunk.SourceStart, chunk.SourceEnd, chunk.CharCount, chunk.TokenEstimate, chunk.ContentHash)
		if err != nil {
			return fmt.Errorf("insert material chunk: %w", err)
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM exam_questions WHERE material_id=?`, j.MaterialID); err != nil {
		return fmt.Errorf("replace exam questions: %w", err)
	}
	for i, question := range questions {
		candidateID := newRepositoryID()
		_, err = tx.ExecContext(ctx, `INSERT INTO exam_question_patterns(id,user_id,learning_space_id,pattern_key,title,description,tested_knowledge,common_mistakes,solving_strategy,occurrence_count,frequency_level) VALUES(?,?,?,?,?,?,?,?,?,0,'low') ON DUPLICATE KEY UPDATE title=VALUES(title),description=VALUES(description),tested_knowledge=VALUES(tested_knowledge),common_mistakes=VALUES(common_mistakes),solving_strategy=VALUES(solving_strategy)`, candidateID, j.UserID, j.LearningSpaceID, question.PatternKey, question.PatternTitle, question.PatternDescription, question.TestedKnowledge, question.CommonMistakes, question.SolvingStrategy)
		if err != nil {
			return fmt.Errorf("upsert exam pattern: %w", err)
		}
		var patternID string
		if err = tx.GetContext(ctx, &patternID, `SELECT id FROM exam_question_patterns WHERE learning_space_id=? AND pattern_key=?`, j.LearningSpaceID, question.PatternKey); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM exam_pattern_node_links WHERE pattern_id=?`, patternID); err != nil {
			return err
		}
		for _, knowledgeName := range question.KnowledgeNames {
			articleBody := examKnowledgeArticleBody(question)
			nodeDescription := strings.TrimSpace(question.TestedKnowledge)
			if nodeDescription == "" {
				nodeDescription = strings.TrimSpace(question.PatternDescription)
			}
			if nodeDescription == "" {
				nodeDescription = "由真题题型分析识别的关联知识点"
			}
			var knowledgeNodeID string
			err = tx.GetContext(ctx, &knowledgeNodeID, `SELECT id FROM learning_nodes WHERE user_id=? AND learning_space_id=? AND name=? ORDER BY CASE WHEN node_type='knowledge' THEN 0 ELSE 1 END,created_at LIMIT 1`, j.UserID, j.LearningSpaceID, knowledgeName)
			if errors.Is(err, sql.ErrNoRows) {
				knowledgeNodeID = newRepositoryID()
				var knowledgeOrder int
				_ = tx.GetContext(ctx, &knowledgeOrder, `SELECT COALESCE(MAX(sort_order),-1)+1 FROM learning_nodes WHERE user_id=? AND learning_space_id=?`, j.UserID, j.LearningSpaceID)
				var sourceChunk any
				if len(chunks) > 0 {
					sourceChunk = chunks[0].ID
				}
				if _, err = tx.ExecContext(ctx, `INSERT INTO learning_nodes(id,user_id,learning_space_id,name,node_type,description,exam_weight,estimated_minutes,source,source_chunk_id,sort_order) VALUES(?,?,?,?,'knowledge',?,0.750,20,'ai',?,?)`, knowledgeNodeID, j.UserID, j.LearningSpaceID, knowledgeName, nodeDescription, sourceChunk, knowledgeOrder); err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `INSERT INTO knowledge_articles(id,user_id,learning_space_id,node_id,title,body,source_chunk_id,sort_order) VALUES(?,?,?,?,?,?,?,?)`, newRepositoryID(), j.UserID, j.LearningSpaceID, knowledgeNodeID, knowledgeName, articleBody, sourceChunk, knowledgeOrder); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			// Upgrade articles created by older versions without overwriting a
			// generated study-material article or anything the user edited.
			if _, err = tx.ExecContext(ctx, `UPDATE knowledge_articles SET body=? WHERE node_id=? AND user_edited=FALSE AND body=?`, articleBody, knowledgeNodeID, legacyExamKnowledgeArticleBody); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE learning_nodes SET description=? WHERE id=? AND user_edited=FALSE AND description='由真题题型分析识别的关联知识点'`, nodeDescription, knowledgeNodeID); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO exam_pattern_node_links(pattern_id,node_id,confidence,relation_reason,source) VALUES(?,?,0.900,?,'ai')`, patternID, knowledgeNodeID, fmt.Sprintf("题型“%s”的考察知识包含“%s”", question.PatternTitle, knowledgeName)); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO exam_questions(id,user_id,learning_space_id,material_id,pattern_id,sequence_no,stem,question_type,source_type,source_start) VALUES(?,?,?,?,?,?,?,?,?,?)`, newRepositoryID(), j.UserID, j.LearningSpaceID, j.MaterialID, patternID, i+1, question.Stem, question.QuestionType, question.SourceType, question.SourceStart)
		if err != nil {
			return fmt.Errorf("insert exam question: %w", err)
		}
	}
	if len(questions) > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE exam_question_patterns p SET occurrence_count=(SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id),frequency_level=CASE WHEN (SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id)>=5 THEN 'high' WHEN (SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id)>=2 THEN 'medium' ELSE 'low' END WHERE p.learning_space_id=?`, j.LearningSpaceID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE p FROM exam_question_patterns p LEFT JOIN exam_questions q ON q.pattern_id=p.id WHERE p.learning_space_id=? AND q.id IS NULL`, j.LearningSpaceID); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO generation_jobs(id,user_id,learning_space_id,material_id,job_type,status,progress,attempts,max_attempts,available_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, embeddingJob.ID, embeddingJob.UserID, embeddingJob.LearningSpaceID, embeddingJob.MaterialID, embeddingJob.JobType, embeddingJob.Status, embeddingJob.Progress, embeddingJob.Attempts, embeddingJob.MaxAttempts, embeddingJob.AvailableAt)
	if err != nil {
		return fmt.Errorf("insert embedding job: %w", err)
	}
	for _, embedding := range embeddings {
		_, err = tx.ExecContext(ctx, `INSERT INTO chunk_embeddings(id,user_id,learning_space_id,material_id,chunk_id,qdrant_point_id,embedding_model,vector_dimension,content_hash,status) VALUES(?,?,?,?,?,?,?,?,?,'pending')`, embedding.ID, embedding.UserID, embedding.LearningSpaceID, embedding.MaterialID, embedding.ChunkID, embedding.QdrantPointID, embedding.EmbeddingModel, embedding.VectorDimension, embedding.ContentHash)
		if err != nil {
			return fmt.Errorf("insert pending chunk embedding: %w", err)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE generation_jobs SET status='succeeded',progress=100,completed_at=? WHERE id=? AND status='processing'`, now, j.ID)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE learning_materials SET status='ready',failure_reason=NULL WHERE id=?`, j.MaterialID)
	if err != nil {
		return fmt.Errorf("complete material: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit complete job: %w", err)
	}
	return nil
}

func examKnowledgeArticleBody(question ai.GeneratedExamQuestion) string {
	summary := strings.TrimSpace(question.TestedKnowledge)
	if summary == "" {
		summary = strings.TrimSpace(question.PatternDescription)
	}
	if summary == "" {
		summary = "该知识点是这类题目的核心理论基础，需要理解定义、适用条件与关键机制。"
	}
	patternTitle := strings.TrimSpace(question.PatternTitle)
	if patternTitle == "" {
		patternTitle = "相关题型"
	}
	patternDescription := strings.TrimSpace(question.PatternDescription)
	if patternDescription == "" {
		patternDescription = "题目会结合具体情境检验对该知识点的理解和应用。"
	}
	strategy := strings.TrimSpace(question.SolvingStrategy)
	if strategy == "" {
		strategy = "先明确核心概念和适用条件，再结合题干信息建立完整的推理过程。"
	}

	return fmt.Sprintf("## 知识点概括\n\n%s\n\n## 在真题中如何考察\n\n**相关题型：** %s\n\n%s\n\n## 理解与应用\n\n%s", summary, patternTitle, patternDescription, strategy)
}

func (r *MaterialRepository) CompleteEmbeddingJob(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE generation_jobs SET status='succeeded',progress=100,completed_at=? WHERE id=? AND job_type='embed_material' AND status='processing'`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("complete embedding job: %w", err)
	}
	return requireAffected(result, "embedding job")
}
func (r *MaterialRepository) UpdateJobProgress(ctx context.Context, id string, progress int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE generation_jobs SET progress=? WHERE id=? AND status='processing'`, progress, id)
	if err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}
	return nil
}
func (r *MaterialRepository) CountChunks(ctx context.Context, userID, spaceID, materialID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM material_chunks WHERE user_id=? AND learning_space_id=? AND material_id=?`, userID, spaceID, materialID)
	if err != nil {
		return 0, fmt.Errorf("count chunks: %w", err)
	}
	return count, nil
}
func (r *MaterialRepository) ListChunks(ctx context.Context, userID, spaceID, materialID string) ([]model.MaterialChunk, error) {
	items := make([]model.MaterialChunk, 0)
	err := r.db.SelectContext(ctx, &items, `SELECT `+chunkColumns+` FROM material_chunks WHERE user_id=? AND learning_space_id=? AND material_id=? ORDER BY chunk_index`, userID, spaceID, materialID)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	return items, nil
}

type RetrievalChunk struct {
	model.MaterialChunk
	MaterialName string `db:"material_name"`
}

func (r *MaterialRepository) GetRetrievalChunks(ctx context.Context, userID, spaceID string, chunkIDs []string) ([]RetrievalChunk, error) {
	if len(chunkIDs) == 0 {
		return []RetrievalChunk{}, nil
	}
	query, args, err := sqlx.In(`SELECT mc.`+strings.ReplaceAll(chunkColumns, ",", ",mc.")+`,lm.original_name AS material_name
        FROM material_chunks mc JOIN learning_materials lm ON lm.id=mc.material_id
        WHERE mc.user_id=? AND mc.learning_space_id=? AND mc.id IN (?)`, userID, spaceID, chunkIDs)
	if err != nil {
		return nil, fmt.Errorf("build retrieval chunk query: %w", err)
	}
	items := make([]RetrievalChunk, 0, len(chunkIDs))
	if err = r.db.SelectContext(ctx, &items, r.db.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("get retrieval chunks: %w", err)
	}
	return items, nil
}

func (r *MaterialRepository) ValidateMaterials(ctx context.Context, userID, spaceID string, materialIDs []string) error {
	if len(materialIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`SELECT COUNT(DISTINCT id) FROM learning_materials WHERE user_id=? AND learning_space_id=? AND id IN (?)`, userID, spaceID, materialIDs)
	if err != nil {
		return fmt.Errorf("build material validation query: %w", err)
	}
	var count int
	if err = r.db.GetContext(ctx, &count, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("validate retrieval materials: %w", err)
	}
	if count != len(materialIDs) {
		return ErrNotFound
	}
	return nil
}

func (r *MaterialRepository) RetryMaterial(ctx context.Context, material model.LearningMaterial, job model.GenerationJob) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin material retry: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE learning_materials SET status='queued',failure_reason=NULL WHERE id=? AND user_id=? AND learning_space_id=? AND status='failed'`, material.ID, material.UserID, material.LearningSpaceID)
	if err != nil {
		return fmt.Errorf("queue material retry: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect material retry: %w", err)
	}
	if count == 0 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO generation_jobs(id,user_id,learning_space_id,material_id,job_type,status,progress,attempts,max_attempts,available_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, job.ID, job.UserID, job.LearningSpaceID, job.MaterialID, job.JobType, job.Status, job.Progress, job.Attempts, job.MaxAttempts, job.AvailableAt)
	if err != nil {
		return fmt.Errorf("insert material retry job: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit material retry: %w", err)
	}
	return nil
}

func (r *MaterialRepository) RetryEmbedding(ctx context.Context, material model.LearningMaterial, job model.GenerationJob, embeddingModel string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin embedding retry: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE chunk_embeddings SET status='pending',failure_reason=NULL,indexed_at=NULL WHERE user_id=? AND learning_space_id=? AND material_id=? AND embedding_model=? AND status='failed'`, material.UserID, material.LearningSpaceID, material.ID, embeddingModel)
	if err != nil {
		return fmt.Errorf("queue embedding records retry: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect embedding retry: %w", err)
	}
	if count == 0 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO generation_jobs(id,user_id,learning_space_id,material_id,job_type,status,progress,attempts,max_attempts,available_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, job.ID, job.UserID, job.LearningSpaceID, job.MaterialID, job.JobType, job.Status, job.Progress, job.Attempts, job.MaxAttempts, job.AvailableAt)
	if err != nil {
		return fmt.Errorf("insert embedding retry job: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit embedding retry: %w", err)
	}
	return nil
}

func (r *MaterialRepository) DeleteMaterial(ctx context.Context, userID, spaceID, materialID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM learning_materials WHERE id=? AND user_id=? AND learning_space_id=?`, materialID, userID, spaceID)
	if err != nil {
		return fmt.Errorf("delete material: %w", err)
	}
	return requireAffected(result, "material")
}

func (r *MaterialRepository) DeleteExamMaterial(ctx context.Context, userID, spaceID, materialID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete exam material: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM learning_materials WHERE id=? AND user_id=? AND learning_space_id=? AND material_kind='past_exam'`, materialID, userID, spaceID)
	if err != nil {
		return fmt.Errorf("delete exam material: %w", err)
	}
	if count, affectedErr := result.RowsAffected(); affectedErr != nil {
		return fmt.Errorf("inspect exam material deletion: %w", affectedErr)
	} else if count == 0 {
		return ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `UPDATE exam_question_patterns p SET
		occurrence_count=(SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id),
		frequency_level=CASE
			WHEN (SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id)>=5 THEN 'high'
			WHEN (SELECT COUNT(*) FROM exam_questions q WHERE q.pattern_id=p.id)>=2 THEN 'medium'
			ELSE 'low' END
		WHERE p.user_id=? AND p.learning_space_id=?`, userID, spaceID)
	if err != nil {
		return fmt.Errorf("recount exam patterns after material deletion: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM exam_question_patterns WHERE user_id=? AND learning_space_id=? AND occurrence_count=0`, userID, spaceID); err != nil {
		return fmt.Errorf("delete orphan exam patterns: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit exam material deletion: %w", err)
	}
	return nil
}

func (r *MaterialRepository) CountKnowledgeReferences(ctx context.Context, userID, spaceID, materialID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM learning_nodes ln JOIN material_chunks mc ON mc.id=ln.source_chunk_id WHERE ln.user_id=? AND ln.learning_space_id=? AND mc.material_id=?`, userID, spaceID, materialID)
	if err != nil {
		return 0, fmt.Errorf("count material knowledge references: %w", err)
	}
	return count, nil
}
func (r *MaterialRepository) FailJob(ctx context.Context, j model.GenerationJob, code, message string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fail job: %w", err)
	}
	defer tx.Rollback()
	terminal := j.Attempts >= j.MaxAttempts
	if terminal {
		now := time.Now().UTC()
		_, err = tx.ExecContext(ctx, `UPDATE generation_jobs SET status='failed',progress=100,completed_at=?,error_code=?,error_message=? WHERE id=?`, now, code, message, j.ID)
		if err == nil && j.JobType == "process_material" {
			_, err = tx.ExecContext(ctx, `UPDATE learning_materials SET status='failed',failure_reason=? WHERE id=?`, message, j.MaterialID)
		}
	} else {
		delay := time.Duration(1<<uint(j.Attempts-1)) * time.Minute
		_, err = tx.ExecContext(ctx, `UPDATE generation_jobs SET status='queued',progress=0,available_at=?,error_code=?,error_message=? WHERE id=?`, time.Now().UTC().Add(delay), code, message, j.ID)
		if err == nil && j.JobType == "process_material" {
			_, err = tx.ExecContext(ctx, `UPDATE learning_materials SET status='queued',failure_reason=? WHERE id=?`, message, j.MaterialID)
		}
	}
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit fail job: %w", err)
	}
	return nil
}
