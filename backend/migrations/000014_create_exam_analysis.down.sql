DROP TABLE IF EXISTS exam_question_feedback;
DROP TABLE IF EXISTS exam_questions;
DROP TABLE IF EXISTS exam_question_patterns;
ALTER TABLE learning_materials DROP CHECK chk_material_kind, DROP COLUMN material_kind;
