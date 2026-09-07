export type AssetJobStatus = 'queued' | 'processing' | 'succeeded' | 'failed'
export interface LearningAssetJob { id:string; learning_space_id:string; job_type:'knowledge_assets'|'learning_plan'; status:AssetJobStatus; progress:number; attempts:number; max_attempts:number; started_at:string|null; completed_at:string|null; error_code:string|null; error_message:string|null; created_at:string; updated_at:string }
export interface LearningNode { id:string; learning_space_id:string; name:string; node_type:string; description:string; exam_weight:number; estimated_minutes:number; source_chunk_id:string|null; sort_order:number; position_x:number|null; position_y:number|null; user_edited:boolean; mastery_score:number; mastery_status:'unassessed'|'weak'|'learning'|'mastered'; evidence_count:number; mastery_confidence:number }
export interface LearningEdge { id:string; from_node_id:string; to_node_id:string; relation_type:string; user_edited:boolean }
export interface KnowledgeNodeInput { name:string; node_type:string; description:string; exam_weight:number; estimated_minutes:number; position_x:number|null; position_y:number|null }
export interface KnowledgeEdgeInput { from_node_id:string; to_node_id:string; relation_type:string }
export interface KnowledgeArticle { id:string; node_id:string; title:string; body:string; source_chunk_id:string|null; sort_order:number; user_edited:boolean }
export interface LearningPlan { id:string; plan_date:string; title:string; total_minutes:number; generation_reason:string; created_at:string }
export interface PlanStage { id:string; plan_id:string; focus_node_id:string; title:string; description:string; status:'pending'|'active'|'complete'; estimated_days:number; sort_order:number }
export interface PlanNode { id:string; plan_id:string; node_id:string; task_type:string; title:string; estimated_minutes:number; sort_order:number }
export interface LearningAssets { knowledge_job:LearningAssetJob|null; plan_job:LearningAssetJob|null; knowledge_map:{nodes:LearningNode[];edges:LearningEdge[]}; handbook:{articles:KnowledgeArticle[]}; today_plan:{plan:LearningPlan|null;stages:PlanStage[];tasks:PlanNode[]} }
