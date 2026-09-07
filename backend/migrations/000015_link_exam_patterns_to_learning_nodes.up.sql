ALTER TABLE exam_question_patterns
  ADD COLUMN learning_node_id CHAR(36) NULL AFTER learning_space_id,
  ADD CONSTRAINT fk_exam_pattern_node FOREIGN KEY(learning_node_id) REFERENCES learning_nodes(id) ON DELETE SET NULL;
