export interface ExamPatternNode { node_id:string; node_name:string; confidence:number; relation_reason:string }
export interface ExamPattern { id:string; pattern_key:string; title:string; description:string; tested_knowledge:string; common_mistakes:string; solving_strategy:string; occurrence_count:number; frequency_level:'low'|'medium'|'high'; related_nodes:ExamPatternNode[] }
export interface ExamQuestion { id:string; material_id:string; pattern_id:string; sequence_no:number; stem:string; question_type:string; source_type:string; source_start:number|null; is_correct:boolean|null; created_at:string }
export interface ExamAnalysis { patterns:ExamPattern[]; questions:ExamQuestion[]; paper_count:number; answered_count:number; wrong_count:number }
export interface ExamFeedbackResult { question_id:string; is_correct:boolean; decision_status:'replanning'|'pending_plan' }
