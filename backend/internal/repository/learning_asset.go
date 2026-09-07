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
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
)

const assetJobColumns = `id,user_id,learning_space_id,job_type,status,progress,attempts,max_attempts,available_at,started_at,completed_at,error_code,error_message,created_at,updated_at`

type LearningAssetRepository struct{ db *sqlx.DB }

func NewLearningAssetRepository(db *sqlx.DB) *LearningAssetRepository {
	return &LearningAssetRepository{db: db}
}

func (r *LearningAssetRepository) CreateJob(ctx context.Context, job model.LearningAssetJob) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin asset job: %w", err)
	}
	defer tx.Rollback()
	var active int
	if err = tx.GetContext(ctx, &active, `SELECT COUNT(*) FROM learning_asset_jobs WHERE user_id=? AND learning_space_id=? AND status IN ('queued','processing') FOR UPDATE`, job.UserID, job.LearningSpaceID); err != nil {
		return fmt.Errorf("check active asset job: %w", err)
	}
	if active > 0 {
		return ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_asset_jobs(id,user_id,learning_space_id,job_type,status,progress,attempts,max_attempts,available_at) VALUES(?,?,?,?,'queued',0,0,?,?)`, job.ID, job.UserID, job.LearningSpaceID, job.JobType, job.MaxAttempts, job.AvailableAt)
	if err != nil {
		return fmt.Errorf("insert asset job: %w", err)
	}
	return tx.Commit()
}
func (r *LearningAssetRepository) GetJob(ctx context.Context, userID, spaceID, id string) (model.LearningAssetJob, error) {
	var j model.LearningAssetJob
	err := r.db.GetContext(ctx, &j, `SELECT `+assetJobColumns+` FROM learning_asset_jobs WHERE id=? AND user_id=? AND learning_space_id=?`, id, userID, spaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}
func (r *LearningAssetRepository) GetJobByID(ctx context.Context, id string) (model.LearningAssetJob, error) {
	var j model.LearningAssetJob
	err := r.db.GetContext(ctx, &j, `SELECT `+assetJobColumns+` FROM learning_asset_jobs WHERE id=?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}
func (r *LearningAssetRepository) LatestJob(ctx context.Context, userID, spaceID string) (model.LearningAssetJob, error) {
	var j model.LearningAssetJob
	err := r.db.GetContext(ctx, &j, `SELECT `+assetJobColumns+` FROM learning_asset_jobs WHERE user_id=? AND learning_space_id=? ORDER BY created_at DESC LIMIT 1`, userID, spaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}
func (r *LearningAssetRepository) LatestJobByType(ctx context.Context, userID, spaceID, jobType string) (model.LearningAssetJob, error) {
	var j model.LearningAssetJob
	err := r.db.GetContext(ctx, &j, `SELECT `+assetJobColumns+` FROM learning_asset_jobs WHERE user_id=? AND learning_space_id=? AND job_type=? ORDER BY created_at DESC LIMIT 1`, userID, spaceID, jobType)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}
func (r *LearningAssetRepository) NextQueuedJobID(ctx context.Context) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id, `SELECT id FROM learning_asset_jobs WHERE status='queued' AND available_at<=? AND attempts<max_attempts ORDER BY available_at,created_at LIMIT 1`, time.Now().UTC())
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
func (r *LearningAssetRepository) ClaimJob(ctx context.Context, id string) (model.LearningAssetJob, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.LearningAssetJob{}, false, err
	}
	defer tx.Rollback()
	var j model.LearningAssetJob
	if err = tx.GetContext(ctx, &j, `SELECT `+assetJobColumns+` FROM learning_asset_jobs WHERE id=? FOR UPDATE`, id); err != nil {
		return j, false, err
	}
	now := time.Now().UTC()
	if j.Status != "queued" || j.AvailableAt.After(now) || j.Attempts >= j.MaxAttempts {
		return j, false, nil
	}
	_, err = tx.ExecContext(ctx, `UPDATE learning_asset_jobs SET status='processing',progress=10,attempts=attempts+1,started_at=?,error_code=NULL,error_message=NULL WHERE id=?`, now, id)
	if err != nil {
		return j, false, err
	}
	if err = tx.Commit(); err != nil {
		return j, false, err
	}
	j.Status = "processing"
	j.Attempts++
	return j, true, nil
}

func (r *LearningAssetRepository) IndexedSources(ctx context.Context, userID, spaceID string) ([]ai.AssetSource, error) {
	items := []ai.AssetSource{}
	err := r.db.SelectContext(ctx, &items, `SELECT mc.id AS chunk_id,mc.content FROM material_chunks mc JOIN chunk_embeddings ce ON ce.chunk_id=mc.id WHERE mc.user_id=? AND mc.learning_space_id=? AND ce.status='indexed' ORDER BY mc.created_at,mc.chunk_index`, userID, spaceID)
	return items, err
}
func (r *LearningAssetRepository) CountIndexedSources(ctx context.Context, userID, spaceID string) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM chunk_embeddings WHERE user_id=? AND learning_space_id=? AND status='indexed'`, userID, spaceID)
	return n, err
}
func (r *LearningAssetRepository) CountUserEditedNodes(ctx context.Context, userID, spaceID string) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `SELECT (SELECT COUNT(*) FROM learning_nodes WHERE user_id=? AND learning_space_id=? AND user_edited=TRUE)+(SELECT COUNT(*) FROM learning_edges WHERE user_id=? AND learning_space_id=? AND user_edited=TRUE)`, userID, spaceID, userID, spaceID)
	return n, err
}

func (r *LearningAssetRepository) CreateNode(ctx context.Context, userID, spaceID string, input dto.KnowledgeNodeInput) (model.LearningNode, error) {
	var sortOrder int
	_ = r.db.GetContext(ctx, &sortOrder, `SELECT COALESCE(MAX(sort_order),-1)+1 FROM learning_nodes WHERE user_id=? AND learning_space_id=?`, userID, spaceID)
	n := model.LearningNode{ID: newRepositoryID(), LearningSpaceID: spaceID, Name: input.Name, NodeType: input.NodeType, Description: input.Description, ExamWeight: input.ExamWeight, EstimatedMinutes: input.EstimatedMinutes, SortOrder: sortOrder, PositionX: input.PositionX, PositionY: input.PositionY, UserEdited: true}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return n, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_nodes(id,user_id,learning_space_id,name,node_type,description,exam_weight,estimated_minutes,source,sort_order,position_x,position_y,user_edited) VALUES(?,?,?,?,?,?,?,?,'user',?,?,?,TRUE)`, n.ID, userID, spaceID, n.Name, n.NodeType, n.Description, n.ExamWeight, n.EstimatedMinutes, n.SortOrder, n.PositionX, n.PositionY)
	if err != nil {
		return n, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO knowledge_articles(id,user_id,learning_space_id,node_id,title,body,sort_order,user_edited) VALUES(?,?,?,?,?,?,?,TRUE)`, newRepositoryID(), userID, spaceID, n.ID, n.Name, n.Description, n.SortOrder)
	if err != nil {
		return n, err
	}
	return n, tx.Commit()
}
func (r *LearningAssetRepository) UpdateNode(ctx context.Context, userID, spaceID, nodeID string, input dto.KnowledgeNodeInput) (model.LearningNode, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE learning_nodes SET name=?,node_type=?,description=?,exam_weight=?,estimated_minutes=?,position_x=?,position_y=?,user_edited=TRUE WHERE id=? AND user_id=? AND learning_space_id=?`, input.Name, input.NodeType, input.Description, input.ExamWeight, input.EstimatedMinutes, input.PositionX, input.PositionY, nodeID, userID, spaceID)
	if err != nil {
		return model.LearningNode{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.LearningNode{}, ErrNotFound
	}
	_, _ = r.db.ExecContext(ctx, `UPDATE knowledge_articles SET title=?,user_edited=TRUE WHERE node_id=? AND user_id=? AND learning_space_id=?`, input.Name, nodeID, userID, spaceID)
	var out model.LearningNode
	err = r.db.GetContext(ctx, &out, `SELECT id,learning_space_id,name,node_type,description,exam_weight,estimated_minutes,source_chunk_id,sort_order,position_x,position_y,user_edited FROM learning_nodes WHERE id=?`, nodeID)
	return out, err
}
func (r *LearningAssetRepository) UpdateNodePosition(ctx context.Context, userID, spaceID, nodeID string, x, y float64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE learning_nodes SET position_x=?,position_y=? WHERE id=? AND user_id=? AND learning_space_id=?`, x, y, nodeID, userID, spaceID)
	if err == nil {
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
	}
	return err
}
func (r *LearningAssetRepository) DeleteNode(ctx context.Context, userID, spaceID, nodeID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM learning_plans WHERE user_id=? AND learning_space_id=?`, userID, spaceID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM learning_nodes WHERE id=? AND user_id=? AND learning_space_id=?`, nodeID, userID, spaceID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}
func (r *LearningAssetRepository) CreateEdge(ctx context.Context, userID, spaceID string, input dto.KnowledgeEdgeInput) (model.LearningEdge, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM learning_nodes WHERE user_id=? AND learning_space_id=? AND id IN (?,?)`, userID, spaceID, input.FromNodeID, input.ToNodeID); err != nil || count != 2 {
		if err != nil {
			return model.LearningEdge{}, err
		}
		return model.LearningEdge{}, ErrNotFound
	}
	e := model.LearningEdge{ID: newRepositoryID(), FromNodeID: input.FromNodeID, ToNodeID: input.ToNodeID, RelationType: input.RelationType, UserEdited: true}
	_, err := r.db.ExecContext(ctx, `INSERT INTO learning_edges(id,user_id,learning_space_id,from_node_id,to_node_id,relation_type,user_edited) VALUES(?,?,?,?,?,?,TRUE)`, e.ID, userID, spaceID, e.FromNodeID, e.ToNodeID, e.RelationType)
	return e, err
}
func (r *LearningAssetRepository) DeleteEdge(ctx context.Context, userID, spaceID, edgeID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM learning_edges WHERE id=? AND user_id=? AND learning_space_id=?`, edgeID, userID, spaceID)
	if err == nil {
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
	}
	return err
}

func (r *LearningAssetRepository) UpdateArticle(ctx context.Context, userID, spaceID, articleID string, input dto.KnowledgeArticleInput) (model.KnowledgeArticle, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE knowledge_articles SET title=?,body=?,user_edited=TRUE WHERE id=? AND user_id=? AND learning_space_id=?`, input.Title, input.Body, articleID, userID, spaceID)
	if err != nil {
		return model.KnowledgeArticle{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.KnowledgeArticle{}, ErrNotFound
	}
	var out model.KnowledgeArticle
	err = r.db.GetContext(ctx, &out, `SELECT id,node_id,title,body,source_chunk_id,sort_order,user_edited FROM knowledge_articles WHERE id=?`, articleID)
	return out, err
}

func (r *LearningAssetRepository) CompleteKnowledge(ctx context.Context, job model.LearningAssetJob, assets ai.GeneratedAssets) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM learning_plans WHERE user_id=? AND learning_space_id=?`, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	type protectedNode struct {
		ID            string  `db:"id"`
		SourceChunkID *string `db:"source_chunk_id"`
	}
	protected := []protectedNode{}
	if err = tx.SelectContext(ctx, &protected, `SELECT DISTINCT n.id,n.source_chunk_id FROM learning_nodes n LEFT JOIN knowledge_articles a ON a.node_id=n.id LEFT JOIN learning_edges e ON (e.from_node_id=n.id OR e.to_node_id=n.id) AND e.user_edited=TRUE WHERE n.user_id=? AND n.learning_space_id=? AND (n.user_edited=TRUE OR COALESCE(a.user_edited,FALSE)=TRUE OR e.id IS NOT NULL)`, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	protectedByChunk := map[string]string{}
	for _, node := range protected {
		if node.SourceChunkID != nil {
			protectedByChunk[*node.SourceChunkID] = node.ID
		}
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM learning_edges WHERE user_id=? AND learning_space_id=? AND user_edited=FALSE`, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	// Nodes referenced by exam patterns are persistent learning state. Deleting
	// them would cascade-delete exam_pattern_node_links, leaving subsequent
	// question feedback with no node whose mastery can be updated.
	if _, err = tx.ExecContext(ctx, `DELETE n FROM learning_nodes n LEFT JOIN knowledge_articles a ON a.node_id=n.id LEFT JOIN learning_edges e ON (e.from_node_id=n.id OR e.to_node_id=n.id) AND e.user_edited=TRUE LEFT JOIN exam_pattern_node_links epnl ON epnl.node_id=n.id WHERE n.user_id=? AND n.learning_space_id=? AND n.user_edited=FALSE AND COALESCE(a.user_edited,FALSE)=FALSE AND e.id IS NULL AND epnl.node_id IS NULL`, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	nodeIDs := make([]string, len(assets.Nodes))
	for i, n := range assets.Nodes {
		if existingID, ok := protectedByChunk[n.SourceChunkID]; ok {
			nodeIDs[i] = existingID
			continue
		}
		nodeIDs[i] = newRepositoryID()
		_, err = tx.ExecContext(ctx, `INSERT INTO learning_nodes(id,user_id,learning_space_id,name,node_type,description,exam_weight,estimated_minutes,source,source_chunk_id,sort_order) VALUES(?,?,?,?,'knowledge',?,?,?,'ai',?,?)`, nodeIDs[i], job.UserID, job.LearningSpaceID, n.Name, n.Description, n.ExamWeight, n.EstimatedMinutes, n.SourceChunkID, i)
		if err != nil {
			return err
		}
	}
	for _, e := range assets.Edges {
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO learning_edges(id,user_id,learning_space_id,from_node_id,to_node_id,relation_type) VALUES(?,?,?,?,?,?)`, newRepositoryID(), job.UserID, job.LearningSpaceID, nodeIDs[e.From], nodeIDs[e.To], e.RelationType)
		if err != nil {
			return err
		}
	}
	for i, a := range assets.Articles {
		if existingID, ok := protectedByChunk[a.SourceChunkID]; ok && existingID == nodeIDs[a.Node] {
			continue
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO knowledge_articles(id,user_id,learning_space_id,node_id,title,body,source_chunk_id,sort_order) VALUES(?,?,?,?,?,?,?,?)`, newRepositoryID(), job.UserID, job.LearningSpaceID, nodeIDs[a.Node], a.Title, a.Body, a.SourceChunkID, i)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE learning_asset_jobs SET status='succeeded',progress=100,completed_at=? WHERE id=? AND status='processing'`, time.Now().UTC(), job.ID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *LearningAssetRepository) PlanSources(ctx context.Context, userID, spaceID string) ([]ai.PlanSource, error) {
	items := []ai.PlanSource{}
	err := r.db.SelectContext(ctx, &items, `SELECT ln.id AS node_id,ln.name,ln.description,ln.estimated_minutes,ln.exam_weight,ka.body AS article_body,COALESCE(lm.original_name,'') AS material_name,COALESCE(s.mastery_score,0) AS mastery_score,COALESCE(s.mastery_status,'unassessed') AS mastery_status,COALESCE(s.evidence_count,0) AS evidence_count,COALESCE(s.confidence,0) AS confidence,COALESCE(s.correct_count,0) AS correct_count,COALESCE(s.wrong_count,0) AS wrong_count FROM learning_nodes ln JOIN knowledge_articles ka ON ka.node_id=ln.id LEFT JOIN material_chunks mc ON mc.id=ln.source_chunk_id LEFT JOIN learning_materials lm ON lm.id=mc.material_id LEFT JOIN user_node_states s ON s.user_id=ln.user_id AND s.node_id=ln.id WHERE ln.user_id=? AND ln.learning_space_id=? ORDER BY ln.sort_order`, userID, spaceID)
	return items, err
}

func (r *LearningAssetRepository) PendingPlanSignals(ctx context.Context, userID, spaceID string) ([]ai.PlanSignal, error) {
	items := []ai.PlanSignal{}
	err := r.db.SelectContext(ctx, &items, `SELECT id,question,COALESCE(related_node_id,'') AS related_node_id,1 AS weight FROM chat_learning_signals WHERE user_id=? AND learning_space_id=? UNION ALL SELECT CONCAT(f.id,':',COALESCE(l.node_id,'')),q.stem,COALESCE(l.node_id,''),CASE WHEN f.is_correct THEN -1 ELSE 3 END AS weight FROM exam_question_feedback f JOIN exam_questions q ON q.id=f.question_id LEFT JOIN exam_pattern_node_links l ON l.pattern_id=q.pattern_id WHERE f.user_id=? AND f.learning_space_id=?`, userID, spaceID, userID, spaceID)
	return items, err
}

func (r *LearningAssetRepository) CompletePlan(ctx context.Context, job model.LearningAssetJob, sources []ai.PlanSource, plan ai.GeneratedPlan, signalCount int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM learning_plans WHERE user_id=? AND learning_space_id=?`, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	planID, total := newRepositoryID(), 0
	for _, task := range plan.Tasks {
		total += task.EstimatedMinutes
	}
	title := strings.TrimSpace(plan.Title)
	if title == "" {
		title = "个性化学习计划"
	}
	reason := strings.TrimSpace(plan.Rationale)
	if reason == "" {
		reason = fmt.Sprintf("AI 已结合资料知识点与 %d 条学情信号生成", signalCount)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO learning_plans(id,user_id,learning_space_id,plan_date,title,total_minutes,generation_reason) VALUES(?,?,?,?,?,?,?)`, planID, job.UserID, job.LearningSpaceID, time.Now().UTC().Format("2006-01-02"), title, total, reason); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE chat_learning_signals SET incorporated_plan_id=? WHERE user_id=? AND learning_space_id=? AND incorporated_plan_id IS NULL`, planID, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE exam_question_feedback SET incorporated_plan_id=? WHERE user_id=? AND learning_space_id=? AND incorporated_plan_id IS NULL`, planID, job.UserID, job.LearningSpaceID); err != nil {
		return err
	}
	for i, stage := range plan.Stages {
		if stage.Node < 0 || stage.Node >= len(sources) {
			return fmt.Errorf("roadmap stage references invalid node")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO plan_stages(id,user_id,learning_space_id,plan_id,focus_node_id,title,description,stage_status,estimated_days,sort_order) VALUES(?,?,?,?,?,?,?,?,?,?)`, newRepositoryID(), job.UserID, job.LearningSpaceID, planID, sources[stage.Node].NodeID, stage.Title, stage.Description, stage.Status, stage.EstimatedDays, i); err != nil {
			return err
		}
	}
	for i, task := range plan.Tasks {
		if task.Node < 0 || task.Node >= len(sources) {
			return fmt.Errorf("plan task references invalid node")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO plan_nodes(id,user_id,learning_space_id,plan_id,node_id,task_type,title,estimated_minutes,sort_order) VALUES(?,?,?,?,?,?,?,?,?)`, newRepositoryID(), job.UserID, job.LearningSpaceID, planID, sources[task.Node].NodeID, task.TaskType, task.Title, task.EstimatedMinutes, i); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE learning_asset_jobs SET status='succeeded',progress=100,completed_at=? WHERE id=? AND status='processing'`, time.Now().UTC(), job.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *LearningAssetRepository) Fail(ctx context.Context, j model.LearningAssetJob, code, message string) error {
	terminal := j.Attempts >= j.MaxAttempts
	if terminal {
		_, err := r.db.ExecContext(ctx, `UPDATE learning_asset_jobs SET status='failed',progress=100,completed_at=?,error_code=?,error_message=? WHERE id=?`, time.Now().UTC(), code, message, j.ID)
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE learning_asset_jobs SET status='queued',progress=0,available_at=?,error_code=?,error_message=? WHERE id=?`, time.Now().UTC().Add(time.Minute), code, message, j.ID)
	return err
}

func (r *LearningAssetRepository) Snapshot(ctx context.Context, userID, spaceID string) (dto.LearningAssets, error) {
	out := dto.LearningAssets{KnowledgeMap: dto.KnowledgeMap{Nodes: []model.LearningNode{}, Edges: []model.LearningEdge{}}, Handbook: dto.KnowledgeHandbook{Articles: []model.KnowledgeArticle{}}, TodayPlan: dto.TodayPlan{Stages: []model.PlanStage{}, Tasks: []model.PlanNode{}}}
	j, err := r.LatestJobByType(ctx, userID, spaceID, "knowledge_assets")
	if err == nil {
		out.KnowledgeJob = &j
	} else if err != ErrNotFound {
		return out, err
	}
	j, err = r.LatestJobByType(ctx, userID, spaceID, "learning_plan")
	if err == nil {
		out.PlanJob = &j
	} else if err != ErrNotFound {
		return out, err
	}
	if err = r.db.SelectContext(ctx, &out.KnowledgeMap.Nodes, `SELECT n.id,n.learning_space_id,n.name,n.node_type,n.description,n.exam_weight,n.estimated_minutes,n.source_chunk_id,n.sort_order,n.position_x,n.position_y,n.user_edited,COALESCE(s.mastery_score,0) AS mastery_score,COALESCE(s.mastery_status,'unassessed') AS mastery_status,COALESCE(s.evidence_count,0) AS evidence_count,COALESCE(s.confidence,0) AS mastery_confidence FROM learning_nodes n LEFT JOIN user_node_states s ON s.node_id=n.id AND s.user_id=n.user_id WHERE n.user_id=? AND n.learning_space_id=? ORDER BY n.sort_order`, userID, spaceID); err != nil {
		return out, err
	}
	if err = r.db.SelectContext(ctx, &out.KnowledgeMap.Edges, `SELECT id,from_node_id,to_node_id,relation_type,user_edited FROM learning_edges WHERE user_id=? AND learning_space_id=?`, userID, spaceID); err != nil {
		return out, err
	}
	if err = r.db.SelectContext(ctx, &out.Handbook.Articles, `SELECT id,node_id,title,body,source_chunk_id,sort_order,user_edited FROM knowledge_articles WHERE user_id=? AND learning_space_id=? ORDER BY sort_order`, userID, spaceID); err != nil {
		return out, err
	}
	var p model.LearningPlan
	err = r.db.GetContext(ctx, &p, `SELECT id,DATE_FORMAT(plan_date,'%Y-%m-%d') AS plan_date,title,total_minutes,generation_reason,created_at FROM learning_plans WHERE user_id=? AND learning_space_id=? ORDER BY created_at DESC LIMIT 1`, userID, spaceID)
	if err == nil {
		out.TodayPlan.Plan = &p
		err = r.db.SelectContext(ctx, &out.TodayPlan.Stages, `SELECT id,plan_id,focus_node_id,title,description,stage_status,estimated_days,sort_order FROM plan_stages WHERE plan_id=? ORDER BY sort_order`, p.ID)
		if err == nil {
			err = r.db.SelectContext(ctx, &out.TodayPlan.Tasks, `SELECT id,plan_id,node_id,task_type,title,estimated_minutes,sort_order FROM plan_nodes WHERE plan_id=? ORDER BY sort_order`, p.ID)
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return out, err
}
