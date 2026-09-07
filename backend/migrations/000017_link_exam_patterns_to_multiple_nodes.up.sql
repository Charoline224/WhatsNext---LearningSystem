CREATE TABLE exam_pattern_node_links (
  pattern_id CHAR(36) NOT NULL,
  node_id CHAR(36) NOT NULL,
  confidence DECIMAL(4,3) NOT NULL,
  relation_reason VARCHAR(500) NOT NULL,
  source VARCHAR(20) NOT NULL DEFAULT 'ai',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY(pattern_id,node_id),
  KEY idx_pattern_node_links_node(node_id),
  CONSTRAINT fk_pattern_node_links_pattern FOREIGN KEY(pattern_id) REFERENCES exam_question_patterns(id) ON DELETE CASCADE,
  CONSTRAINT fk_pattern_node_links_node FOREIGN KEY(node_id) REFERENCES learning_nodes(id) ON DELETE CASCADE
);

INSERT INTO exam_pattern_node_links(pattern_id,node_id,confidence,relation_reason,source)
SELECT id,learning_node_id,1.000,'由原单节点关联迁移','ai'
FROM exam_question_patterns WHERE learning_node_id IS NOT NULL;

ALTER TABLE exam_question_patterns
  DROP FOREIGN KEY fk_exam_pattern_node,
  DROP COLUMN learning_node_id;
