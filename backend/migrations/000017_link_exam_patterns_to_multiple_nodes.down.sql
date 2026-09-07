ALTER TABLE exam_question_patterns
  ADD COLUMN learning_node_id CHAR(36) NULL AFTER learning_space_id,
  ADD CONSTRAINT fk_exam_pattern_node FOREIGN KEY(learning_node_id) REFERENCES learning_nodes(id) ON DELETE SET NULL;
UPDATE exam_question_patterns p SET learning_node_id=(SELECT l.node_id FROM exam_pattern_node_links l WHERE l.pattern_id=p.id ORDER BY l.confidence DESC LIMIT 1);
DROP TABLE IF EXISTS exam_pattern_node_links;
