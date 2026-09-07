DELETE FROM learning_nodes
WHERE source = 'ai'
  AND node_type = 'practice'
  AND user_edited = FALSE
  AND name LIKE '题型：%';
