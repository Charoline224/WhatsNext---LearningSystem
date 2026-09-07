ALTER TABLE learning_materials
  ADD COLUMN material_kind VARCHAR(20) NOT NULL DEFAULT 'study' AFTER learning_space_id,
  ADD CONSTRAINT chk_material_kind CHECK (material_kind IN ('study','past_exam'));

CREATE TABLE exam_question_patterns (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  learning_space_id CHAR(36) NOT NULL,
  pattern_key VARCHAR(120) NOT NULL,
  title VARCHAR(200) NOT NULL,
  description TEXT NOT NULL,
  occurrence_count INT UNSIGNED NOT NULL DEFAULT 0,
  frequency_level VARCHAR(10) NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  UNIQUE KEY uk_exam_pattern_space (learning_space_id,pattern_key),
  CONSTRAINT fk_exam_pattern_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_pattern_space FOREIGN KEY(learning_space_id) REFERENCES learning_spaces(id) ON DELETE CASCADE
);

CREATE TABLE exam_questions (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  learning_space_id CHAR(36) NOT NULL,
  material_id CHAR(36) NOT NULL,
  pattern_id CHAR(36) NOT NULL,
  sequence_no INT UNSIGNED NOT NULL,
  stem MEDIUMTEXT NOT NULL,
  question_type VARCHAR(30) NOT NULL,
  source_type VARCHAR(20) NOT NULL,
  source_start INT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uk_exam_question_sequence(material_id,sequence_no),
  KEY idx_exam_questions_space(learning_space_id,created_at),
  CONSTRAINT fk_exam_question_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_question_space FOREIGN KEY(learning_space_id) REFERENCES learning_spaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_question_material FOREIGN KEY(material_id) REFERENCES learning_materials(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_question_pattern FOREIGN KEY(pattern_id) REFERENCES exam_question_patterns(id) ON DELETE CASCADE
);

CREATE TABLE exam_question_feedback (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  learning_space_id CHAR(36) NOT NULL,
  question_id CHAR(36) NOT NULL,
  is_correct BOOLEAN NOT NULL,
  note VARCHAR(500) NOT NULL DEFAULT '',
  incorporated_plan_id CHAR(36) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  UNIQUE KEY uk_exam_feedback_user_question(user_id,question_id),
  KEY idx_exam_feedback_pending(user_id,learning_space_id,incorporated_plan_id),
  CONSTRAINT fk_exam_feedback_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_feedback_space FOREIGN KEY(learning_space_id) REFERENCES learning_spaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_feedback_question FOREIGN KEY(question_id) REFERENCES exam_questions(id) ON DELETE CASCADE,
  CONSTRAINT fk_exam_feedback_plan FOREIGN KEY(incorporated_plan_id) REFERENCES learning_plans(id) ON DELETE SET NULL
);
