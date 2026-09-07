CREATE TABLE chat_learning_signals (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NOT NULL,
  learning_space_id CHAR(36) NOT NULL,
  question TEXT NOT NULL,
  answer MEDIUMTEXT NOT NULL,
  sources_json JSON NOT NULL,
  related_node_id CHAR(36) NULL,
  incorporated_plan_id CHAR(36) NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  INDEX idx_chat_signals_pending (user_id, learning_space_id, incorporated_plan_id, created_at),
  CONSTRAINT fk_chat_signals_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_chat_signals_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_chat_signals_node FOREIGN KEY (related_node_id) REFERENCES learning_nodes(id) ON DELETE SET NULL,
  CONSTRAINT fk_chat_signals_plan FOREIGN KEY (incorporated_plan_id) REFERENCES learning_plans(id) ON DELETE SET NULL
);
